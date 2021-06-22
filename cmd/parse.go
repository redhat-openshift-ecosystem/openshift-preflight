package cmd

import (
	"os"
	"strings"

	"github.com/komish/preflight/certification/runtime"
)

// getStringConfigValueFrom will return a value from the options if they're
// not empty. Precedence is flagValue, envValue, defaultValue.
func getConfigStringValueFrom(flagValue, envValue, defaultValue string) string {
	if len(flagValue) > 0 {
		return flagValue
	}

	if len(envValue) > 0 {
		return envValue
	}

	return defaultValue
}

func parseEnabledPoliciesValue() []string {
	val := getConfigStringValueFrom(userEnabledPolicies, os.Getenv(EnvEnabledPolicies), "")
	if len(val) == 0 {
		return runtime.GetPoliciesByName()
	}

	return strings.Split(val, ",")
}

func parseOutputFormat() string {
	return getConfigStringValueFrom(userOutputFormat, os.Getenv(EnvOutputFormat), defaultOutputFormat)
}
