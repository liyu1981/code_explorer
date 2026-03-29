package agent

func GetIntegrationTestParams() (string, string, string, string, bool) {
	stype := "openai"
	baseURL := "http://localhost:20003/v1"
	model := "unsloth/Qwen3.5-9B-GGUF:Q4_K_M"
	apiKey := ""
	noThink := true

	return stype, baseURL, model, apiKey, noThink
}
