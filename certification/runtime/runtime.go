// Package runtime contains the structs and definitions consumed by Preflight at
// runtime.
package runtime

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
