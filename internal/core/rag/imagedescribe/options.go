package imagedescribe

import (
	"github.com/boxify/api-go/internal/core/jsonx"
	coreprompt "github.com/boxify/api-go/internal/core/prompt"
	"github.com/boxify/api-go/internal/core/rag/imagecompress"
	ragprompt "github.com/boxify/api-go/internal/core/rag/prompt"
)

const (
	defaultMaxTokens = int64(1024)
)

var defaultPrompt = coreprompt.MustRender(ragprompt.Templates, ragprompt.ImageDescriptionTemplate, nil)

// Options 定义 Describer 的长期配置。
type Options struct {
	Prompt     string
	MaxTokens  int64
	Compressor Compressor
	Parser     jsonx.Parser
}

// Option 修改 Describer 的长期配置。
type Option func(*Options)

// WithPrompt 设置发送给视觉模型的最终提示词文本。
func WithPrompt(prompt string) Option {
	return func(opts *Options) {
		if prompt != "" {
			opts.Prompt = prompt
		}
	}
}

// WithMaxTokens 设置视觉模型最大输出 token 数。
func WithMaxTokens(maxTokens int64) Option {
	return func(opts *Options) {
		if maxTokens > 0 {
			opts.MaxTokens = maxTokens
		}
	}
}

// WithCompressor 设置图片压缩器。
func WithCompressor(compressor Compressor) Option {
	return func(opts *Options) {
		if compressor != nil {
			opts.Compressor = compressor
		}
	}
}

// WithParser 设置模型 JSON 输出解析器。
func WithParser(parser jsonx.Parser) Option {
	return func(opts *Options) {
		if parser != nil {
			opts.Parser = parser
		}
	}
}

// defaultCompressor 返回图片描述默认使用的压缩器。
func defaultCompressor() Compressor {
	return imagecompress.NewCompressor()
}

// defaultParser 返回图片描述默认使用的 JSON 解析器。
func defaultParser() jsonx.Parser {
	return jsonx.NewParser()
}
