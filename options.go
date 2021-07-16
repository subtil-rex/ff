package ff

// Context contains private fields used during parsing.
type Context struct {
	configFileVia              *string
	configFileFlagName         string
	configFileParser           ConfigFileParser
	allowMissingConfigFile     bool
	envVarPrefix               string
	envVarNoPrefix             bool
	envVarSplit                string
	ignoreUndefinedConfigFlags bool
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
//
// Deprecated: use WithIgnoreUndefinedConfigFlags.
func WithIgnoreUndefined(ignore bool) Option {
	return func(c *Context) {
		c.ignoreUndefinedConfigFlags = ignore
	}
}

// WithIgnoreUndefinedConfigFlags tells Parse to ignore any flags that it
// encounters in config files that aren't defined in the flag set. By default,
// undefined flags in config files are treated as parse errors.
//
// Note that this setting only applies to config files. Passing an undefined
// flag as a commandline arugment will always be an error.
func WithIgnoreUndefinedConfigFlags() Option {
	return func(c *Context) {
		c.ignoreUndefinedConfigFlags = true
	}
}
