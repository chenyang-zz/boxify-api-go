package search

import (
	"context"
)

// ESClient 定义检索器需要的最小 Elasticsearch 查询能力。
type ESClient interface {
	Search(ctx context.Context, index string, query any) (map[string]any, error)
}

// Embedder 定义单文本向量化能力。
type Embedder interface {
	EmbedOne(ctx context.Context, text string, dimensions int) ([]float64, error)
}

// Reranker 定义候选文档重排能力。
//
// RerankResult.Index 必须指向 documents 中的下标；越界结果会被忽略。
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

// FilterBuilder 根据请求构造 ES filter。
//
// 返回的 filter 会同时用于向量召回和 BM25 召回；业务过滤规则由调用方自行实现。
type FilterBuilder func(ctx context.Context, req Input) ([]any, error)

// SourceDecoder 把 ES _source 解码成调用方需要的类型。
//
// decoder 返回错误时 Search 会终止并返回该错误，避免静默丢失业务元数据。
type SourceDecoder[T any] func(src map[string]any) (T, error)

// RerankResult 表示重排模型返回的候选下标和分数。
type RerankResult struct {
	Index int
	Score float64
}

// Input 描述一次 RAG 检索请求的内部配置。
// RecallSize 控制两路召回池大小，最终仍由 TopK 裁剪。
// Filters 会透传给向量召回和 BM25 召回，业务过滤规则由调用方提供。
// MinVectorScore 启用后按 ES cosine 原始相关度门控，适合精确搜索场景。
type Input struct {
	Query          string
	TopK           int
	RecallSize     int
	Filters        []any
	MinVectorScore *float64
	embedder       Embedder
}

// InputOption 修改单次 Search 请求配置。
type InputOption func(*Input)

// WithTopK 设置最终返回结果数量。
func WithTopK(topK int) InputOption {
	return func(req *Input) {
		if topK > 0 {
			req.TopK = topK
		}
	}
}

// WithInputRecallSize 设置单次请求的两路召回池大小。
func WithInputRecallSize(recallSize int) InputOption {
	return func(req *Input) {
		if recallSize > 0 {
			req.RecallSize = recallSize
		}
	}
}

// WithFilters 设置单次请求要透传给 ES 的 filter。
func WithFilters(filters []any) InputOption {
	return func(req *Input) {
		req.Filters = filters
	}
}

// WithMinVectorScore 设置最低 cosine 向量相关度门槛。
func WithMinVectorScore(minVectorScore float64) InputOption {
	return func(req *Input) {
		req.MinVectorScore = &minVectorScore
	}
}

// WithInputEmbedder 设置单次请求使用的向量化客户端。
func WithInputEmbedder(embedder Embedder) InputOption {
	return func(req *Input) {
		if embedder != nil {
			req.embedder = embedder
		}
	}
}

// Output 表示一次检索命中的通用结果。
//
// ID 是 ES hit 的 _id，Content 优先使用 parent chunk 内容，Source 是调用方 decoder 的输出。
type Output[T any] struct {
	ID      string
	Content string
	Score   float64
	Source  T
}
