package cmd

var (
	EnvEnabledChecks = "PREFLIGHT_ENABLED_CHECKS"
	EnvOutputFormat  = "PREFLIGHT_OUTPUT_FORMAT"
	EnvCLILogFile    = "PREFLIGHT_CLI_LOG_FILE"
)

var (
	defaultOutputFormat   = "json"
	defaultCLILogFileName = "preflight.log"
)
