package container

import (
	"fmt"
	"regexp"
	"strings"
)

// violatesRedHatTrademark validates if a string meets specific "Red Hat" naming criteria
func violatesRedHatTrademark(s string) (bool, error) {
	// string starts with Red Hat variant
	startingWithRedHatRegexp, err := regexp.Compile("^[^a-z0-9]*red[^a-z0-9]*hat")
	if err != nil {
		return false, fmt.Errorf("error while compiling regexp: %w", err)
	}
	startingWithRedHat := startingWithRedHatRegexp.MatchString(strings.ToLower(s))

	// string contain Red Hat variant (not starting with)
	containsRedHatRegexp, err := regexp.Compile("red[^a-z0-9]*hat")
	if err != nil {
		return false, fmt.Errorf("error while compiling regexp: %w", err)
	}
	containsRedHat := len(containsRedHatRegexp.FindAllString(strings.ToLower(s), -1))

	// string contains "for Red Hat" variant
	containsForRedHatRegexp, err := regexp.Compile("for[^a-z0-9]*red[^a-z0-9]*hat")
	if err != nil {
		return false, fmt.Errorf("error while compiling regexp: %w", err)
	}
	containsForRedHat := containsForRedHatRegexp.MatchString(strings.ToLower(s))

	// We explicitly fail for this, so we don't need to count it here.
	if startingWithRedHat {
		containsRedHat -= 1
	}

	// This is acceptable, so we don't count it against the string.
	if containsForRedHat {
		containsRedHat -= 1
	}

	containsInvalidReference := containsRedHat > 0

	return startingWithRedHat || containsInvalidReference, nil
}
