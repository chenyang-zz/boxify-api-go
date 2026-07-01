package search

import (
	"context"
)

type ESClient interface {
	Search(ctx context.Context, index string, query any) (map[string]any, error)
}

type Embedder interface {
	EmbedOne(ctx context.Context, text string, dimensions int) ([]float64, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

type FilterBuilder func(ctx context.Context, req Request) ([]any, error)

type SourceDecoder[T any] func(src map[string]any) (T, error)

type RerankResult struct {
	Index int
	Score float64
}

// Request 描述一次 RAG 检索请求的内部配置。
// RecallSize 控制两路召回池大小，最终仍由 TopK 裁剪。
// Filters 会透传给向量召回和 BM25 召回，业务过滤规则由调用方提供。
// MinVectorScore 启用后按 ES cosine 原始相关度门控，适合精确搜索场景。
type Request struct {
	Query          string
	TopK           int
	RecallSize     int
	Filters        []any
	MinVectorScore *float64
}

type RequestOption func(*Request)

func WithTopK(topK int) RequestOption {
	return func(req *Request) {
		if topK > 0 {
			req.TopK = topK
		}
	}
}

func WithRequestRecallSize(recallSize int) RequestOption {
	return func(req *Request) {
		if recallSize > 0 {
			req.RecallSize = recallSize
		}
	}
}

func WithFilters(filters []any) RequestOption {
	return func(req *Request) {
		req.Filters = filters
	}
}

func WithMinVectorScore(minVectorScore float64) RequestOption {
	return func(req *Request) {
		req.MinVectorScore = &minVectorScore
	}
}

type Result[T any] struct {
	ID      string
	Content string
	Score   float64
	Source  T
}
