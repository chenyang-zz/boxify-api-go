package types_test

import (
	"testing"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/google/uuid"
)

// 验证业务任务名称顺序稳定，worker 注册和调度任务不会因为 map 迭代产生抖动。
func TestTaskNamesAreStable(t *testing.T) {
	names := types.TaskNames()
	want := []types.TaskName{
		types.TaskParseDocument,
		types.TaskParseImage,
		types.TaskMemoryExtract,
		types.TaskMemoryConsolidate,
		types.TaskResearchRun,
	}
	if len(names) != len(want) {
		t.Fatalf("names = %#v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

// 验证文档解析任务由 domain 类型包统一构造，并带上 parse 队列和强类型 payload。
func TestNewParseDocumentTaskBuildsTypedDomainTask(t *testing.T) {
	userID := uuid.New()
	documentID := uuid.New()

	task, err := types.NewParseDocumentTask(userID, documentID)
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	if task.Name != types.TaskParseDocument {
		t.Fatalf("task name = %q, want %q", task.Name, types.TaskParseDocument)
	}
	if task.Queue != types.QueueParse {
		t.Fatalf("task queue = %q, want %q", task.Queue, types.QueueParse)
	}
	payload, ok := task.Payload.(*types.ParseDocumentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *types.ParseDocumentPayload", task.Payload)
	}
	if payload.UserID != userID || payload.DocumentID != documentID {
		t.Fatalf("payload = %+v, want user/document ids", payload)
	}
}

// 验证文档解析任务会拒绝空 UUID，避免无效任务进入队列。
func TestNewParseDocumentTaskRejectsNilIDs(t *testing.T) {
	if _, err := types.NewParseDocumentTask(uuid.Nil, uuid.New()); err == nil {
		t.Fatal("NewParseDocumentTask user nil error = nil, want error")
	}
	if _, err := types.NewParseDocumentTask(uuid.New(), uuid.Nil); err == nil {
		t.Fatal("NewParseDocumentTask document nil error = nil, want error")
	}
}
