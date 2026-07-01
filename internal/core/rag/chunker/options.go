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

type Options struct {
	ChildChunkTokens  int
	ParentChunkTokens int
	ChildOverlapRatio float64
	SentenceRegex     *regexp.Regexp
	TokenEncodingName string
}

type Option func(*Options)

func WithChildChunkTokens(childChunkTokens int) Option {
	return func(opts *Options) {
		opts.ChildChunkTokens = childChunkTokens
	}
}

func WithParentChunkTokens(parentChunkTokens int) Option {
	return func(opts *Options) {
		opts.ParentChunkTokens = parentChunkTokens
	}
}

func WithChildOverlapRatio(childOverlapRatio float64) Option {
	return func(opts *Options) {
		opts.ChildOverlapRatio = childOverlapRatio
	}
}

func WithSentenceRegex(sentenceRegex *regexp.Regexp) Option {
	return func(opts *Options) {
		opts.SentenceRegex = sentenceRegex
	}
}

func WithTokenEncodingName(tokenEncodingName string) Option {
	return func(opts *Options) {
		opts.TokenEncodingName = tokenEncodingName
	}
}
