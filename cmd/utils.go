package cmd

import "strings"

func resultsFilenameWithExtension(ext string) string {
	return strings.Join([]string{"results", ext}, ".")
}
