package chunker

import "regexp"

const (
	defaultChildChunkTokens  = 512
	defaultParentChunkTokens = 1024
	defaultChildOverlapRatio = 0.1
	defaultTokenEncodingName = "cl100k_base"
)

var defaultSentenceRegex = regexp.MustCompile(`[^。！？.!?\n]+[。！？.!?\n]?`)

// Options 定义 Chunker 的长期分块配置。
//
// ChildChunkTokens 控制子块 token 上限，ParentChunkTokens 控制父块 token 上限。
// ChildOverlapRatio 控制子块之间保留的句子重叠比例；SentenceRegex 控制句子切分规则。
type Options struct {
	ChildChunkTokens  int
	ParentChunkTokens int
	ChildOverlapRatio float64
	SentenceRegex     *regexp.Regexp
	TokenEncodingName string
}

// Option 修改 Chunker 的长期分块配置。
type Option func(*Options)

// WithChildChunkTokens 设置子块 token 上限。
func WithChildChunkTokens(childChunkTokens int) Option {
	return func(opts *Options) {
		opts.ChildChunkTokens = childChunkTokens
	}
}

// WithParentChunkTokens 设置父块 token 上限。
func WithParentChunkTokens(parentChunkTokens int) Option {
	return func(opts *Options) {
		opts.ParentChunkTokens = parentChunkTokens
	}
}

// WithChildOverlapRatio 设置子块之间的句子重叠比例。
func WithChildOverlapRatio(childOverlapRatio float64) Option {
	return func(opts *Options) {
		opts.ChildOverlapRatio = childOverlapRatio
	}
}

// WithSentenceRegex 设置句子切分正则。
func WithSentenceRegex(sentenceRegex *regexp.Regexp) Option {
	return func(opts *Options) {
		opts.SentenceRegex = sentenceRegex
	}
}

// WithTokenEncodingName 设置 tiktoken 编码名称。
func WithTokenEncodingName(tokenEncodingName string) Option {
	return func(opts *Options) {
		opts.TokenEncodingName = tokenEncodingName
	}
}
