package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAnswer       = "Local chat reply persisted."
	defaultInitialDelay = 750 * time.Millisecond
	defaultChunkDelay   = 350 * time.Millisecond
)

type chatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type server struct {
	answer       string
	initialDelay time.Duration
	chunkDelay   time.Duration
}

func main() {
	address := flag.String("address", envOrDefault("COVE_E2E_LLM_ADDRESS", "127.0.0.1:58001"), "listen address")
	answer := flag.String("answer", envOrDefault("COVE_E2E_LLM_ANSWER", defaultAnswer), "deterministic assistant answer")
	flag.Parse()

	initialDelay, err := envDuration("COVE_E2E_LLM_INITIAL_DELAY", defaultInitialDelay)
	if err != nil {
		log.Fatal(err)
	}
	chunkDelay, err := envDuration("COVE_E2E_LLM_CHUNK_DELAY", defaultChunkDelay)
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*answer) == "" {
		log.Fatal("answer must not be empty")
	}

	handler := newHandler(server{
		answer:       strings.TrimSpace(*answer),
		initialDelay: initialDelay,
		chunkDelay:   chunkDelay,
	})
	log.Printf("deterministic OpenAI-compatible provider listening on %s", *address)
	if err := http.ListenAndServe(*address, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newHandler(provider server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /v1/chat/completions", provider.chatCompletions)
	return mux
}

func (s server) chatCompletions(w http.ResponseWriter, request *http.Request) {
	var input chatCompletionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, `{"error":{"message":"invalid JSON request"}}`, http.StatusBadRequest)
		return
	}
	if !input.Stream {
		http.Error(w, `{"error":{"message":"stream=true is required"}}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Model) == "" {
		http.Error(w, `{"error":{"message":"model is required"}}`, http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if !wait(request, s.initialDelay) {
		return
	}
	for _, chunk := range splitAnswer(s.answer) {
		payload := map[string]any{
			"id":      "chatcmpl-cove-e2e",
			"object":  "chat.completion.chunk",
			"created": 1,
			"model":   input.Model,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]string{"content": chunk},
			}},
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
			return
		}
		flusher.Flush()
		if !wait(request, s.chunkDelay) {
			return
		}
	}
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func splitAnswer(answer string) []string {
	runes := []rune(answer)
	if len(runes) < 3 {
		return []string{answer}
	}
	first := len(runes) / 3
	second := first * 2
	return []string{string(runes[:first]), string(runes[first:second]), string(runes[second:])}
}

func wait(request *http.Request, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-request.Context().Done():
		return false
	case <-timer.C:
		return true
	}
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative duration: %q", name, raw)
	}
	return value, nil
}

func envOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
