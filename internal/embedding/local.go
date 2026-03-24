package embedding

import (
	"net/http"
	"strings"
	"time"
)

// NewLocalClient creates a Client for a local embedding server that does not
// require an API key (e.g. Ollama, llama.cpp). A placeholder key is used so
// the Authorization header is sent; local servers typically ignore it.
// Returns nil if baseURL is empty.
func NewLocalClient(baseURL, model string) *Client {
	if baseURL == "" {
		return nil
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "nomic-embed-text"
	}

	provider := "local"
	if strings.Contains(baseURL, "11434") || strings.Contains(baseURL, "ollama") {
		provider = "ollama"
	}

	return &Client{
		baseURL:  baseURL,
		model:    model,
		apiKey:   "local", // placeholder; local servers ignore auth
		http:     &http.Client{Timeout: 10 * time.Second},
		provider: provider,
	}
}

// ProbeOllama checks whether Ollama is running on localhost:11434.
// Returns a Client configured to use it, or nil if Ollama is not reachable.
func ProbeOllama() *Client {
	probe := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := probe.Get("http://localhost:11434")
	if err != nil {
		return nil
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil
	}
	return NewLocalClient("http://localhost:11434/v1", "nomic-embed-text")
}
