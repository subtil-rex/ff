package ff

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Parse the flags in the flag set from the provided (presumably commandline)
// args. Additional options may be provided to parse from a config file and/or
// environment variables in that priority order.
func Parse(fs *flag.FlagSet, args []string, options ...Option) error {
	var c Context
	for _, option := range options {
		option(&c)
	}

	// First priority: commandline flags (explicit user preference).
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing commandline args: %w", err)
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Second priority: environment variables (session).
	if parseEnv := c.envVarPrefix != "" || c.envVarNoPrefix; parseEnv {
		var visitErr error
		fs.VisitAll(func(f *flag.Flag) {
			if visitErr != nil {
				return
			}

			if provided[f.Name] {
				return
			}

			var key string
			key = strings.ToUpper(f.Name)
			key = envVarReplacer.Replace(key)
			key = maybePrefix(key, c.envVarNoPrefix, c.envVarPrefix)

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

	var configFile string
	if c.configFileVia != nil {
		configFile = *c.configFileVia
	}

	// Third priority: config file (host).
	if configFile == "" && c.configFileFlagName != "" {
		if f := fs.Lookup(c.configFileFlagName); f != nil {
			configFile = f.Value.String()
		}
	}

	if parseConfig := configFile != "" && c.configFileParser != nil; parseConfig {
		f, err := os.Open(configFile)
		switch {
		case err == nil:
			defer f.Close()
			if err := c.configFileParser(f, func(name, value string) error {
				if provided[name] {
					return nil
				}

				defined := fs.Lookup(name) != nil
				switch {
				case !defined && c.ignoreUndefinedConfigFlags:
					return nil
				case !defined && !c.ignoreUndefinedConfigFlags:
					return fmt.Errorf("config file flag %q not defined in flag set", name)
				}

				if err := fs.Set(name, value); err != nil {
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

var envVarReplacer = strings.NewReplacer(
	"-", "_",
	".", "_",
	"/", "_",
)

func maybePrefix(key string, noPrefix bool, prefix string) string {
	if noPrefix {
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
