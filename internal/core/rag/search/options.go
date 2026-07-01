package search

const (
	defaultIndex        = "comet_chunks"
	defaultVectorWeight = 0.6
	defaultBM25Weight   = 0.4
	defaultTopK         = 5
	defaultRecallSize   = 20
	defaultEmbeddingDim = 1024
)

type Options struct {
	Index         string
	EmbeddingDim  int
	RecallSize    int
	VectorWeight  float64
	BM25Weight    float64
	KnnOversample int
	Reranker      Reranker
	FilterBuilder FilterBuilder
	sourceDecoder any
}

type Option func(*Options)

func WithIndex(index string) Option {
	return func(opts *Options) {
		if index != "" {
			opts.Index = index
		}
	}
}

func WithEmbeddingDim(embeddingDim int) Option {
	return func(opts *Options) {
		if embeddingDim > 0 {
			opts.EmbeddingDim = embeddingDim
		}
	}
}

func WithRecallSize(recallSize int) Option {
	return func(opts *Options) {
		if recallSize > 0 {
			opts.RecallSize = recallSize
		}
	}
}

func WithVectorWeight(vectorWeight float64) Option {
	return func(opts *Options) {
		opts.VectorWeight = vectorWeight
	}
}

func WithBM25Weight(bm25Weight float64) Option {
	return func(opts *Options) {
		opts.BM25Weight = bm25Weight
	}
}

func WithKnnOversample(knnOversample int) Option {
	return func(opts *Options) {
		if knnOversample > 0 {
			opts.KnnOversample = knnOversample
		}
	}
}

func WithReranker(reranker Reranker) Option {
	return func(opts *Options) {
		opts.Reranker = reranker
	}
}

func WithFilterBuilder(builder FilterBuilder) Option {
	return func(opts *Options) {
		if builder != nil {
			opts.FilterBuilder = builder
		}
	}
}

func WithSourceDecoder[T any](decoder SourceDecoder[T]) Option {
	return func(opts *Options) {
		if decoder != nil {
			opts.sourceDecoder = decoder
		}
	}
}
