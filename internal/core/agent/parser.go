package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	coretool "github.com/boxify/api-go/internal/core/tool"
)

var (
	reactThoughtRE     = regexp.MustCompile(`(?m)^Thought\s*:\s*(.+)$`)
	reactActionRE      = regexp.MustCompile(`(?m)^Action\s*:\s*(.+)$`)
	reactActionInputRE = regexp.MustCompile(`(?s)Action\s*Input\s*:\s*(.*)$`)
	reactFinalRE       = regexp.MustCompile(`(?s)Final\s*Answer\s*:\s*(.*)$`)
)

// Parser 解析模型输出为 Agent 可执行的决策。
type Parser interface {
	Parse(ctx context.Context, text string) (Decision, error)
}

// ReActParser 解析基础 ReAct 文本协议。
type ReActParser struct{}

// NewReActParser 创建默认 ReAct parser。
func NewReActParser() *ReActParser {
	return &ReActParser{}
}

// Parse 解析 ReAct 文本输出。
func (p *ReActParser) Parse(ctx context.Context, text string) (Decision, error) {
	thought := firstMatch(reactThoughtRE, text)
	if final := firstMatch(reactFinalRE, text); final != "" {
		return Decision{
			Kind:        DecisionFinal,
			Thought:     thought,
			FinalAnswer: strings.TrimSpace(final),
		}, nil
	}

	action := firstMatch(reactActionRE, text)
	if action == "" {
		return Decision{}, fmt.Errorf("%w: missing action or final answer", ErrParseDecision)
	}
	input, err := parseActionInput(firstMatch(reactActionInputRE, text))
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		Kind:        DecisionToolCall,
		Thought:     thought,
		Action:      strings.TrimSpace(action),
		ActionInput: input,
	}, nil
}

func parseActionInput(raw string) (coretool.Input, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return coretool.Input{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidActionInput, err)
	}
	if out == nil {
		return nil, errors.New("invalid action input")
	}
	return coretool.Input(out), nil
}

func firstMatch(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}
