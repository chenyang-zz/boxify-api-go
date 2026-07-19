package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChatCompletionsStreamsDeterministicAnswer 验证兼容接口会以 SSE 分块返回确定性答案。
func TestChatCompletionsStreamsDeterministicAnswer(t *testing.T) {
	provider := httptest.NewServer(newHandler(server{answer: "deterministic answer"}))
	t.Cleanup(provider.Close)

	response, err := http.Post(
		provider.URL+"/v1/chat/completions",
		"application/json",
		bytes.NewBufferString(`{"model":"cove-e2e","stream":true}`),
	)
	if err != nil {
		t.Fatalf("POST chat completions error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 200; body=%q", response.StatusCode, body)
	}

	var streamed strings.Builder
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil {
			t.Fatalf("decode stream chunk error = %v", err)
		}
		if len(chunk.Choices) != 1 {
			t.Fatalf("choices = %d, want 1", len(chunk.Choices))
		}
		streamed.WriteString(chunk.Choices[0].Delta.Content)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream error = %v", err)
	}
	if streamed.String() != "deterministic answer" {
		t.Fatalf("streamed answer = %q, want deterministic answer", streamed.String())
	}
}
