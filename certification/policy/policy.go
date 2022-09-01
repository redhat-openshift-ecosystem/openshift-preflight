package policy

type Policy = string

const (
	PolicyOperator       Policy = "operator"
	PolicyContainer      Policy = "container"
	PolicyScratchNonRoot Policy = "scratch-nonroot"
	PolicyRoot           Policy = "root"
)
