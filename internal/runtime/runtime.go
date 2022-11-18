// Package runtime contains the structs and definitions consumed by Preflight at
// runtime.
package runtime

import "fmt"

type OpenshiftClusterVersion struct {
	Name    string
	Version string
}

func UnknownOpenshiftClusterVersion() OpenshiftClusterVersion {
	return OpenshiftClusterVersion{
		Name:    "unknown",
		Version: "unknown",
	}
}

func (v OpenshiftClusterVersion) String() string {
	return fmt.Sprintf("%s/%s", v.Name, v.Version)
}
