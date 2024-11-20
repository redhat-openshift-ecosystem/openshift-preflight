package container

import (
	"regexp"
	"strings"
)

// violatesRedHatTrademark validates if a string meets specific "Red Hat" naming criteria
func violatesRedHatTrademark(s string) bool {
	// string starts with Red Hat variant
	startingWithRedHat := regexp.MustCompile("^[^a-z0-9]*red[^a-z0-9]*hat").MatchString(strings.ToLower(s))

	// string contain Red Hat variant (not starting with)
	containsRedHat := len(regexp.MustCompile("red[^a-z0-9]*hat").FindAllString(strings.ToLower(s), -1))

	// string contains "for Red Hat" variant
	containsForRedHat := regexp.MustCompile("for[^a-z0-9]*red[^a-z0-9]*hat").MatchString(strings.ToLower(s))

	// We explicitly fail for this, so we don't need to count it here.
	if startingWithRedHat {
		containsRedHat -= 1
	}

	// This is acceptable, so we don't count it against the string.
	if containsForRedHat {
		containsRedHat -= 1
	}

	containsInvalidReference := containsRedHat > 0

	return startingWithRedHat || containsInvalidReference
}
