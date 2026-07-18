package classifier

import (
	"github.com/boxify/api-go/internal/core/jsonx"
	corellm "github.com/boxify/api-go/internal/core/llm"
	coreprompt "github.com/boxify/api-go/internal/core/prompt"
	ragprompt "github.com/boxify/api-go/internal/core/rag/prompt"
)

const (
	defaultTemperature  = 0.2
	defaultMaxTokens    = int64(200)
	defaultSnippetRunes = 1500
)

var defaultPrompt = coreprompt.MustTemplateText(ragprompt.Templates, ragprompt.ContentClassifierTemplate)

// Options 定义 Classifier 的长期配置。
//
// Prompt 默认是 Go template 文本；通过 WithPrompt 设置后会被视为最终提示词，不再做模板渲染。
type Options struct {
	Prompt       string
	Temperature  float64
	MaxTokens    int64
	SnippetRunes int
	Parser       jsonx.Parser
	client       TextClient
	promptTmpl   bool
}

// Option 修改 Classifier 的长期配置。
type Option func(*Options)

// WithClient 设置分类器默认使用的文本模型客户端。
//
// client 为 nil 时会被忽略，调用方可以在 Classify 时通过 WithInputClient 注入请求级客户端。
func WithClient(client corellm.Client) Option {
	return func(opts *Options) {
		if client != nil {
			opts.client = llmTextClient{client: client}
		}
	}
}

// WithTextClient 设置分类器默认使用的最小文本模型客户端。
//
// client 为 nil 时会被忽略；它和 WithClient 同时使用时，后传的 option 生效。
func WithTextClient(client TextClient) Option {
	return func(opts *Options) {
		if client != nil {
			opts.client = client
		}
	}
}

// WithPrompt 设置外部传入的最终提示词文本。
//
// 非空 prompt 会关闭默认模板渲染，因此不会替换其中的 Go template 参数。
func WithPrompt(prompt string) Option {
	return func(opts *Options) {
		if prompt != "" {
			opts.Prompt = prompt
			opts.promptTmpl = false
		}
	}
}

// WithTemperature 设置模型采样温度。
func WithTemperature(temperature float64) Option {
	return func(opts *Options) {
		if temperature >= 0 {
			opts.Temperature = temperature
		}
	}
}

// WithMaxTokens 设置模型最大输出 token 数。
func WithMaxTokens(maxTokens int64) Option {
	return func(opts *Options) {
		if maxTokens > 0 {
			opts.MaxTokens = maxTokens
		}
	}
}

// WithSnippetRunes 设置放入默认提示词的正文最大 rune 数。
func WithSnippetRunes(snippetRunes int) Option {
	return func(opts *Options) {
		if snippetRunes > 0 {
			opts.SnippetRunes = snippetRunes
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

// WithInputClient 设置单次分类请求使用的文本模型客户端。
//
// 请求级 client 优先于构造级默认 client；client 为 nil 时会被忽略。
func WithInputClient(client corellm.Client) InputOption {
	return func(input *Input) {
		if client != nil {
			input.client = llmTextClient{client: client}
		}
	}
}

// WithInputTextClient 设置单次分类请求使用的最小文本模型客户端。
//
// 请求级 client 优先于构造级默认 client；它和 WithInputClient 同时使用时，后传的 option 生效。
func WithInputTextClient(client TextClient) InputOption {
	return func(input *Input) {
		if client != nil {
			input.client = client
		}
	}
}

// defaultParser 返回分类器默认使用的 JSON 解析器。
func defaultParser() jsonx.Parser {
	return jsonx.NewParser()
}
