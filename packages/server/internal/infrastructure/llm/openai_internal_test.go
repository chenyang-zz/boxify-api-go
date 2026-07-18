package llm

import "testing"

func TestNormalizeOpenAIBaseURLAddsVersionForOfficialHost(t *testing.T) {
	if got := normalizeOpenAIBaseURL(" https://api.openai.com/ "); got != "https://api.openai.com/v1" {
		t.Fatalf("normalizeOpenAIBaseURL = %q, want official /v1 URL", got)
	}
	if got := normalizeOpenAIBaseURL("https://api.openai.com/v1/"); got != "https://api.openai.com/v1" {
		t.Fatalf("normalizeOpenAIBaseURL = %q, want trimmed official /v1 URL", got)
	}
}

func TestNormalizeOpenAIBaseURLPreservesCompatibleProviderPaths(t *testing.T) {
	cases := []string{
		"https://api.deepseek.com",
		"https://dashscope.aliyuncs.com/compatible-mode/v1",
		"http://127.0.0.1:8080",
	}
	for _, tc := range cases {
		if got := normalizeOpenAIBaseURL(tc); got != tc {
			t.Fatalf("normalizeOpenAIBaseURL(%q) = %q, want unchanged", tc, got)
		}
	}
}
