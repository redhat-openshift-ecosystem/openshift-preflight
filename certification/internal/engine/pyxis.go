package engine

type pyxisEngine struct {
	ApiToken string
}

func NewPyxisEngine(apiToken string) *pyxisEngine {
	return &pyxisEngine{
		ApiToken: apiToken,
	}
}
