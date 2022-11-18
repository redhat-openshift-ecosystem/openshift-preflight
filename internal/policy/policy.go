package policy

type Policy = string

const (
	PolicyOperator  Policy = "operator"
	PolicyContainer Policy = "container"
	PolicyScratch   Policy = "scratch"
	PolicyRoot      Policy = "root"
)
