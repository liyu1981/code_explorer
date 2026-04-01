//go:build integration

package tests

func GetIntegrationTestParams() (string, string, string, string, bool, int) {
	stype := "openai"
	baseURL := "http://localhost:20003/v1"
	model := "unsloth/Qwen3.5-9B-GGUF:Q4_K_M"
	apiKey := ""
	noThink := true
	embeddingSize := 4096

	return stype, baseURL, model, apiKey, noThink, embeddingSize
}
