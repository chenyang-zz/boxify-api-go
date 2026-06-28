/**
 * @Time   : 2026/6/23 01:26
 * @Author : chenyangzhao542@gmail.com
 * @File   : manager.go
 **/

package prompt

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/Masterminds/sprig/v3"
	"github.com/boxify/api-go/internal/xerr"
)

type Manager struct {
	root          string
	MemoryPrompts *MemoryPrompts
	AgentPrompts  *AgentPrompts
}

func NewManager(root string) *Manager {
	m := &Manager{
		root: root,
	}
	memoryPrompts := NewMemoryPrompts(m)
	agentPrompts := NewAgentPrompts(m)

	m.MemoryPrompts = memoryPrompts
	m.AgentPrompts = agentPrompts

	return m
}

func (m *Manager) Render(name string, data any) (string, error) {
	path := filepath.Join(m.root, fmt.Sprintf("%s.tmpl", name))

	content, err := os.ReadFile(path)
	if err != nil {
		return "", xerr.Wrapf(err, "read prompt %s failed: %v", path, err)
	}

	tpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(string(content))
	if err != nil {
		return "", xerr.Wrapf(err, "parse prompt %s failed: %v", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", xerr.Wrapf(err, "render prompt %s failed: %v", name, err)
	}

	return buf.String(), nil
}
