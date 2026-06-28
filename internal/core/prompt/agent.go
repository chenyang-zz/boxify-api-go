/**
 * @Time   : 2026/6/28 23:24
 * @Author : chenyangzhao542@gmail.com
 * @File   : agent.go
 **/

package prompt

import "fmt"

type OptimizePromptData struct {
	RawPrompt string
}

type AgentPrompts struct {
	namespace string
	manger    *Manager
}

func NewAgentPrompts(manager *Manager) *AgentPrompts {
	return &AgentPrompts{
		namespace: "agent",
		manger:    manager,
	}
}

func (p *AgentPrompts) OptimizePrompt(data *OptimizePromptData) (string, error) {
	return p._render("optimize_prompt", data)
}

func (p *AgentPrompts) _render(promptName string, data any) (string, error) {
	return p.manger.Render(fmt.Sprintf("%s/%s", p.namespace, promptName), data)
}
