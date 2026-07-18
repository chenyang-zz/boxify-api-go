package classifier

import (
	"context"

	corellm "github.com/boxify/api-go/internal/core/llm"
)

// llmTextClient 把 core llm 客户端适配为分类器需要的单提示词文本客户端。
type llmTextClient struct {
	client corellm.Client
}

// Classify 使用用户消息调用文本模型，并透传分类器的采样参数。
func (c llmTextClient) Classify(ctx context.Context, prompt string, temperature float64, maxTokens int64) (string, error) {
	return c.client.Invoke(ctx, []*corellm.Message{{Role: corellm.UserRole, Content: prompt}}, corellm.WithTemperature(temperature), corellm.WithMaxTokens(maxTokens))
}
