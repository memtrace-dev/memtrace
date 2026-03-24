package embedding

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLocalClient_Defaults(t *testing.T) {
	c := NewLocalClient("http://localhost:11434/v1", "")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.model != "nomic-embed-text" {
		t.Errorf("want default model nomic-embed-text, got %s", c.model)
	}
	if c.provider != "ollama" {
		t.Errorf("want provider ollama, got %s", c.provider)
	}
}

func TestNewLocalClient_EmptyURL(t *testing.T) {
	c := NewLocalClient("", "")
	if c != nil {
		t.Error("expected nil for empty URL")
	}
}

func TestNewLocalClient_CustomModel(t *testing.T) {
	c := NewLocalClient("http://localhost:8080/v1", "mxbai-embed-large")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.model != "mxbai-embed-large" {
		t.Errorf("want mxbai-embed-large, got %s", c.model)
	}
	if c.provider != "local" {
		t.Errorf("want provider local, got %s", c.provider)
	}
}

func TestNewLocalClient_TrailingSlash(t *testing.T) {
	c := NewLocalClient("http://localhost:11434/v1/", "")
	if c.baseURL != "http://localhost:11434/v1" {
		t.Errorf("trailing slash not stripped, got %s", c.baseURL)
	}
}

func TestProbeOllama_Running(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Override the probe to use our test server.
	// ProbeOllama hard-codes localhost:11434, so we test NewLocalClient directly.
	c := NewLocalClient(srv.URL+"/v1", "nomic-embed-text")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.Provider() != "local" {
		t.Errorf("unexpected provider: %s", c.Provider())
	}
	if c.Model() != "nomic-embed-text" {
		t.Errorf("unexpected model: %s", c.Model())
	}
}

func TestClient_Provider_OpenAI(t *testing.T) {
	c := NewClient("sk-test", "", "", "")
	if c.Provider() != "openai" {
		t.Errorf("want openai, got %s", c.Provider())
	}
}

func TestClient_Provider_Ollama(t *testing.T) {
	c := NewClient("k", "", "http://localhost:11434/v1", "nomic-embed-text")
	if c.Provider() != "ollama" {
		t.Errorf("want ollama, got %s", c.Provider())
	}
}

func TestClient_Provider_Custom(t *testing.T) {
	c := NewClient("k", "", "https://my-embed-service.example.com/v1", "my-model")
	if c.Provider() != "custom" {
		t.Errorf("want custom, got %s", c.Provider())
	}
}

func TestClient_Model(t *testing.T) {
	c := NewClient("k", "", "", "text-embedding-ada-002")
	if c.Model() != "text-embedding-ada-002" {
		t.Errorf("want text-embedding-ada-002, got %s", c.Model())
	}
}
