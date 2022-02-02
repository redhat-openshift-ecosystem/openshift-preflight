package operator

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// annotation() accepts the annotations map and searches for the specified annotation corresponding
// with the key, which is then returned.
func annotation(annotations map[string]string, key string) (string, error) {
	log.Tracef("searching for key (%s) in bundle", key)
	log.Trace("bundle data: ", annotations)
	value, found := annotations[key]
	if !found {
		return "", fmt.Errorf("did not find value at the key %s in the annotations.yaml", key)
	}

	return value, nil
}
