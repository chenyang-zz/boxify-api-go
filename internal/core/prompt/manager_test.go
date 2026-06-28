package prompt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/core/prompt"
)

func TestManagerRenderReadsTemplateFromRootPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "memory"), 0o755); err != nil {
		t.Fatalf("mkdir prompt namespace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "memory", "example.tmpl"), []byte("hello {{ .Name }}"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	manager := prompt.NewManager(root)
	got, err := manager.Render("memory/example", map[string]string{"Name": "boxify"})
	if err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if got != "hello boxify" {
		t.Fatalf("Render = %q, want hello boxify", got)
	}
}

func TestManagerRenderMissingTemplateIncludesPath(t *testing.T) {
	root := t.TempDir()
	manager := prompt.NewManager(root)

	_, err := manager.Render("memory/missing", nil)
	if err == nil {
		t.Fatal("Render error = nil, want missing template error")
	}
	want := filepath.Join(root, "memory", "missing.tmpl")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Render error = %q, want path %q", err.Error(), want)
	}
}
