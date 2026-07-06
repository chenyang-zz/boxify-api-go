package llm

import "testing"

// 验证 NewChatOptions 会提供核心层默认温度，并允许调用方显式覆盖。
func TestNewChatOptionsDefaultsTemperatureAndAllowsOverride(t *testing.T) {
	defaults := NewChatOptions()
	if defaults.Temperature == nil {
		t.Fatal("NewChatOptions().Temperature = nil, want default value")
	}
	if *defaults.Temperature != DefaultTemperature {
		t.Fatalf("NewChatOptions().Temperature = %v, want %v", *defaults.Temperature, DefaultTemperature)
	}
	if defaults.MaxTokens != nil {
		t.Fatalf("NewChatOptions().MaxTokens = %v, want nil", *defaults.MaxTokens)
	}

	overridden := NewChatOptions(WithTemperature(0.2))
	if overridden.Temperature == nil || *overridden.Temperature != 0.2 {
		t.Fatalf("NewChatOptions(WithTemperature).Temperature = %v, want 0.2", overridden.Temperature)
	}
}
