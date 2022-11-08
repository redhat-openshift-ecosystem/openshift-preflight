package runtime

func PyxisHostLookup(pyxisEnv, hostOverride string) string {
	envs := map[string]string{
		"prod":  "catalog.redhat.com/api/containers",
		"uat":   "catalog.uat.redhat.com/api/containers",
		"qa":    "catalog.qa.redhat.com/api/containers",
		"stage": "catalog.stage.redhat.com/api/containers",
	}
	if hostOverride != "" {
		return hostOverride
	}

	pyxisHost, ok := envs[pyxisEnv]
	if !ok {
		pyxisHost = envs["prod"]
	}
	return pyxisHost
}
