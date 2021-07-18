package ff

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfigFileParser interprets the config file represented by the reader
// and calls the set function for each parsed flag pair.
type ConfigFileParser func(r io.Reader, set func(name, value string) error) error

type ConfigFileLookup func(fs *flag.FlagSet, name string) *flag.Flag

// Parse the flags in the flag set from the provided (presumably commandline)
// args. Additional options may be provided to parse from a config file and/or
// environment variables in that priority order.
func Parse(fs *flag.FlagSet, args []string, options ...Option) error {
	var c Context
	for _, option := range options {
		option(&c)
	}

	flag2env := map[*flag.Flag]string{}
	env2flag := map[string]*flag.Flag{}
	fs.VisitAll(func(f *flag.Flag) {
		var key string
		key = strings.ToUpper(f.Name)
		key = flagNameToEnvVar.Replace(key)
		key = maybePrefix(key, c.envVarNoPrefix, c.envVarPrefix)
		env2flag[key] = f
		flag2env[f] = key
	})

	// First priority: commandline flags (explicit user preference).
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing commandline args: %w", err)
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Second priority: environment variables (session).
	parseEnv := c.envVarPrefix != "" || c.envVarNoPrefix
	if parseEnv {
		var visitErr error
		fs.VisitAll(func(f *flag.Flag) {
			if visitErr != nil {
				return
			}

			if provided[f.Name] {
				return
			}

			key, ok := flag2env[f]
			if !ok {
				panic(fmt.Errorf("%s: invalid flag/env mapping", f.Name))
			}

			value := os.Getenv(key)
			if value == "" {
				return
			}

			for _, v := range maybeSplit(value, c.envVarSplit) {
				if err := fs.Set(f.Name, v); err != nil {
					visitErr = fmt.Errorf("error setting flag %q from env var %q: %w", f.Name, key, err)
					return
				}
			}
		})
		if visitErr != nil {
			return fmt.Errorf("error parsing env vars: %w", visitErr)
		}
	}

	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Third priority: config file (host).
	var configFile string
	if c.configFileVia != nil {
		configFile = *c.configFileVia
	}

	if configFile == "" && c.configFileFlagName != "" {
		if f := fs.Lookup(c.configFileFlagName); f != nil {
			configFile = f.Value.String()
		}
	}

	if c.configFileLookup == nil {
		c.configFileLookup = func(fs *flag.FlagSet, name string) *flag.Flag {
			return fs.Lookup(name)
		}
	}

	var (
		haveConfigFile  = configFile != ""
		haveParser      = c.configFileParser != nil
		parseConfigFile = haveConfigFile && haveParser
	)
	if parseConfigFile {
		f, err := os.Open(configFile)
		switch {
		case err == nil:
			defer f.Close()
			if err := c.configFileParser(f, func(name, value string) error {
				if provided[name] {
					return nil
				}

				var (
					f1 = fs.Lookup(name)
					f2 = env2flag[name]
					f  *flag.Flag
				)
				switch {
				case f1 == nil && f2 == nil && c.ignoreUndefined:
					return nil
				case f1 == nil && f2 == nil && !c.ignoreUndefined:
					return fmt.Errorf("config file flag %q not defined in flag set", name)
				case f1 != nil && f2 == nil:
					f = f1
				case f1 == nil && f2 != nil:
					f = f2
				case f1 != nil && f2 != nil && f1 == f2:
					f = f1
				case f1 != nil && f2 != nil && f1 != f2:
					return fmt.Errorf("config file flag %q ambiguous: matches %s and %s", name, f1.Name, f2.Name)
				}

				if provided[f.Name] {
					return nil
				}

				if err := fs.Set(f.Name, value); err != nil {
					return fmt.Errorf("error setting flag %q from config file: %w", name, err)
				}

				return nil
			}); err != nil {
				return err
			}

		case os.IsNotExist(err) && c.allowMissingConfigFile:
			// no problem

		default:
			return err
		}
	}

	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	return nil
}

// Context contains private fields used during parsing.
type Context struct {
	configFileVia          *string
	configFileFlagName     string
	configFileParser       ConfigFileParser
	configFileLookup       ConfigFileLookup
	allowMissingConfigFile bool
	envVarPrefix           string
	envVarNoPrefix         bool
	envVarSplit            string
	ignoreUndefined        bool
}

// Option controls some aspect of Parse behavior.
type Option func(*Context)

// WithConfigFile tells Parse to read the provided filename as a config file.
// Requires WithConfigFileParser, and overrides WithConfigFileFlag.
// Because config files should generally be user-specifiable, this option
// should be rarely used. Prefer WithConfigFileFlag.
func WithConfigFile(filename string) Option {
	return WithConfigFileVia(&filename)
}

// WithConfigFileVia tells Parse to read the provided filename as a config file.
// Requires WithConfigFileParser, and overrides WithConfigFileFlag.
// This is useful for sharing a single root level flag for config files among
// multiple ffcli subcommands.
func WithConfigFileVia(filename *string) Option {
	return func(c *Context) {
		c.configFileVia = filename
	}
}

// WithConfigFileFlag tells Parse to treat the flag with the given name as a
// config file. Requires WithConfigFileParser, and is overridden by
// WithConfigFile.
//
// To specify a default config file, provide it as the default value of the
// corresponding flag -- and consider also using the WithAllowMissingConfigFile
// option.
func WithConfigFileFlag(flagname string) Option {
	return func(c *Context) {
		c.configFileFlagName = flagname
	}
}

// WithConfigFileParser tells Parse how to interpret the config file provided
// via WithConfigFile or WithConfigFileFlag.
func WithConfigFileParser(p ConfigFileParser) Option {
	return func(c *Context) {
		c.configFileParser = p
	}
}

// WithAllowMissingConfigFile tells Parse to permit the case where a config file
// is specified but doesn't exist. By default, missing config files result in an
// error.
func WithAllowMissingConfigFile(allow bool) Option {
	return func(c *Context) {
		c.allowMissingConfigFile = allow
	}
}

// WithEnvVarPrefix tells Parse to try to set flags from environment variables
// with the given prefix. Flag names are matched to environment variables with
// the given prefix, followed by an underscore, followed by the capitalized flag
// names, with separator characters like periods or hyphens replaced with
// underscores. By default, flags are not set from environment variables at all.
func WithEnvVarPrefix(prefix string) Option {
	return func(c *Context) {
		c.envVarPrefix = prefix
	}
}

// WithEnvVarNoPrefix tells Parse to try to set flags from environment variables
// without any specific prefix. Flag names are matched to environment variables
// by capitalizing the flag name, and replacing separator characters like
// periods or hyphens with underscores. By default, flags are not set from
// environment variables at all.
func WithEnvVarNoPrefix() Option {
	return func(c *Context) {
		c.envVarNoPrefix = true
	}
}

// WithEnvVarSplit tells Parse to split environment variables on the given
// delimiter, and to make a call to Set on the corresponding flag with each
// split token.
func WithEnvVarSplit(delimiter string) Option {
	return func(c *Context) {
		c.envVarSplit = delimiter
	}
}

// WithIgnoreUndefined tells Parse to ignore undefined flags that it encounters
// in config files. By default, if Parse encounters an undefined flag in a
// config file, it will return an error. Note that this setting does not apply
// to undefined flags passed as arguments.
func WithIgnoreUndefined(ignore bool) Option {
	return func(c *Context) {
		c.ignoreUndefined = ignore
	}
}

func envVarToFlagNames(env string) []string {
	lower := strings.ToLower(env)
	return []string{
		lower,
		strings.ReplaceAll(lower, "_", "-"),
		strings.ReplaceAll(lower, "_", "."),
		strings.ReplaceAll(lower, "_", "/"),
	}
}

var flagNameToEnvVar = strings.NewReplacer(
	"-", "_",
	".", "_",
	"/", "_",
)

func maybePrefix(key string, noPrefix bool, prefix string) string {
	if noPrefix || prefix == "" {
		return key
	}
	return strings.ToUpper(prefix) + "_" + key
}

func maybeSplit(value, split string) []string {
	if split == "" {
		return []string{value}
	}
	return strings.Split(value, split)
}
