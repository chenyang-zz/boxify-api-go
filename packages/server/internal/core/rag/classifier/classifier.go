package classifier

import (
	"context"
	"errors"
	"strings"

	coreprompt "github.com/boxify/api-go/internal/core/prompt"
	ragprompt "github.com/boxify/api-go/internal/core/rag/prompt"
	"github.com/boxify/api-go/internal/core/valuex"
)

// Classifier 使用文本模型为内容生成宽泛标签。
//
// Classifier 不持有业务状态；调用方可以复用同一个实例处理多次分类请求。
type Classifier struct {
	Options
	client TextClient
}

// NewClassifier 创建内容分类器。
//
// 默认不携带文本模型客户端；调用方可以通过 WithClient 设置长期 client，
// 或在 Classify 时通过 WithInputClient 注入请求级 client。
func NewClassifier(opts ...Option) *Classifier {
	classifier := &Classifier{
		Options: Options{
			Prompt:       defaultPrompt,
			Temperature:  defaultTemperature,
			MaxTokens:    defaultMaxTokens,
			SnippetRunes: defaultSnippetRunes,
			Parser:       defaultParser(),
			promptTmpl:   true,
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&classifier.Options)
		}
	}
	classifier.client = classifier.Options.client
	return classifier
}

// Classify 调用文本模型生成最多两个标签。
//
// 模型调用失败或输出解析失败时返回空标签且 error 为 nil，避免分类辅助能力阻断主流程。
// client 或 Parser 未配置时返回错误。
func (c *Classifier) Classify(ctx context.Context, input Input, opts ...InputOption) (*Result, error) {
	for _, opt := range opts {
		if opt != nil {
			opt(&input)
		}
	}
	client := input.client
	if client == nil && c != nil {
		client = c.client
	}
	if c == nil || client == nil {
		return nil, errors.New("rag classifier text client is nil")
	}
	if c.Parser == nil {
		return nil, errors.New("rag classifier json parser is nil")
	}

	prompt, err := c.buildPrompt(input)
	if err != nil {
		return nil, err
	}

	// 分类是辅助能力：模型调用失败不能阻断文档主流程。
	answer, err := client.Classify(ctx, prompt, c.Temperature, c.MaxTokens)
	if err != nil {
		return &Result{Tags: []string{}}, nil
	}
	return &Result{Tags: c.parseTags(answer)}, nil
}

// buildPrompt 根据配置构建最终提示词。
func (c *Classifier) buildPrompt(input Input) (string, error) {
	if !c.promptTmpl {
		return c.Prompt, nil
	}
	existing := "（暂无，可自行创造）"
	if len(input.ExistingTags) > 0 {
		existing = strings.Join(input.ExistingTags, "、")
	}
	content := valuex.TruncateRunes(input.Content, c.SnippetRunes)
	return coreprompt.RenderText(c.Prompt, ragprompt.ContentClassifierData{
		Existing: existing,
		Content:  content,
	})
}

// parseTags 从模型原文中提取并规整标签。
func (c *Classifier) parseTags(answer string) []string {
	text := extractJSONArray(answer)
	if text == "" {
		return []string{}
	}
	var raw []any
	if err := c.Parser.Unmarshal(text, &raw); err != nil {
		return []string{}
	}

	// 模型输出不稳定，这里只保留字符串、去空白、截断并限制数量。
	tags := make([]string, 0, 2)
	for _, item := range raw {
		tag := valuex.TruncateRunes(valuex.String(item), 16)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
		if len(tags) == 2 {
			break
		}
	}
	return tags
}

// extractJSONArray 从 markdown 或混合文本中截取 JSON 数组片段。
func extractJSONArray(answer string) string {
	text := strings.TrimSpace(answer)
	if strings.HasPrefix(text, "```") {
		text = strings.Trim(text, "`")
		text = strings.TrimSpace(strings.TrimPrefix(text, "json"))
	}
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end < start {
		return ""
	}
	return text[start : end+1]
}
