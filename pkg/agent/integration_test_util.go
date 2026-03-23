package agent

func GetIntegrationTestParams() (string, string, string) {
	baseURL := "http://localhost:20003/v1"
	model := "unsloth/Qwen3.5-9B-GGUF:Q4_K_M"
	apiKey := ""

	return baseURL, model, apiKey
}
