/**
 * @Time   : 2026/7/1 17:11
 * @Author : chenyangzhao542@gmail.com
 * @File   : options.go
 **/

package chunker

import "regexp"

const (
	defaultChildChunkTokens  = 512
	defaultParentChunkTokens = 1024
	defaultChildOverlapRatio = 0.1
	defaultTokenEncodingName = "cl100k_base"
)

var defaultSentenceRegex = regexp.MustCompile(`[^。！？.!?\n]+[。！？.!?\n]?`)

type ChunkOptions struct {
	ChildChunkTokens  int
	ParentChunkTokens int
	ChildOverlapRatio float64
	SentenceRegex     *regexp.Regexp
	TokenEncodingName string
}

type ChunkOption func(*ChunkOptions)

func WithChildChunkTokens(childChunkTokens int) ChunkOption {
	return func(opts *ChunkOptions) {
		opts.ChildChunkTokens = childChunkTokens
	}
}

func WithParentChunkTokens(parentChunkTokens int) ChunkOption {
	return func(opts *ChunkOptions) {
		opts.ParentChunkTokens = parentChunkTokens
	}
}

func WithChildOverlapRatio(childOverlapRatio float64) ChunkOption {
	return func(opts *ChunkOptions) {
		opts.ChildOverlapRatio = childOverlapRatio
	}
}

func WithSentenceRegex(sentenceRegex *regexp.Regexp) ChunkOption {
	return func(opts *ChunkOptions) {
		opts.SentenceRegex = sentenceRegex
	}
}

func WithTokenEncodingName(tokenEncodingName string) ChunkOption {
	return func(opts *ChunkOptions) {
		opts.TokenEncodingName = tokenEncodingName
	}
}
