package classifier

import "context"

// TextClient 定义内容分类所需的最小文本模型能力。
//
// Classify 接收最终提示词和生成参数，返回模型原文；classifier 包会把 core llm client 适配到该接口。
type TextClient interface {
	Classify(ctx context.Context, prompt string, temperature float64, maxTokens int64) (string, error)
}

// Input 表示一次内容分类请求。
//
// Content 是待分类正文，ExistingTags 是可供模型优先复用的已有标签。
type Input struct {
	Content      string
	ExistingTags []string
	client       TextClient
}

// Result 表示内容分类结果。
//
// Tags 最多保留两个规整后的标签；模型失败或解析失败时 Tags 为空切片。
type Result struct {
	Tags []string
}

// InputOption 修改单次分类请求配置。
type InputOption func(*Input)
