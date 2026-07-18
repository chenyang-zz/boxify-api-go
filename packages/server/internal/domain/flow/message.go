package flow

type MessageKind string

const (
	MessageAssistant  MessageKind = "assistant"
	MessagePartial    MessageKind = "partial"
	MessageToolCall   MessageKind = "tool_call"
	MessageToolResult MessageKind = "tool_result"
	MessageThink      MessageKind = "think"
	MessageError      MessageKind = "error"
	MessageDone       MessageKind = "done"
)

// Think 状态：模型请求中 / 本轮模型调用结束。
const (
	ThinkStatusThinking = "thinking"
	ThinkStatusDone     = "done"
)

type Message interface {
	Kind() MessageKind
}

type AssistantMessage struct {
	Answer string
}

func (*AssistantMessage) Kind() MessageKind {
	return MessageAssistant
}

type PartialMessage struct {
	// Text 是已经确认可展示的 assistant 文本增量，不包含工具协议或 ReAct 内部字段。
	Text string
}

func (*PartialMessage) Kind() MessageKind {
	return MessagePartial
}

type ToolCallMessage struct {
	Tool       string
	Input      map[string]any
	Iteration  int
	ToolCallID string
}

func (*ToolCallMessage) Kind() MessageKind {
	return MessageToolCall
}

type ToolResultMessage struct {
	Tool        string
	Input       map[string]any
	Observation string
	Error       string
	Iteration   int
	ToolCallID  string
}

func (*ToolResultMessage) Kind() MessageKind {
	return MessageToolResult
}

// ThinkMessage 表示大模型请求状态（瞬时 UI，不落库）。
type ThinkMessage struct {
	Status    string // thinking | done
	Iteration int
}

func (*ThinkMessage) Kind() MessageKind {
	return MessageThink
}

type ErrorMessage struct {
	Message string
	Partial string
	Err     error
}

func (*ErrorMessage) Kind() MessageKind {
	return MessageError
}

type DoneMessage struct{}

func (*DoneMessage) Kind() MessageKind {
	return MessageDone
}
