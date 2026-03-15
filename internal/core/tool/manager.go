package tool

import "fmt"

// Manager 负责集中管理工具实例，供 node 按名称调用。
type Manager struct {
	tools map[string]Tool
}

// NewManager 依据已注册工具构建管理器。
func NewManager(toolNames []string) (*Manager, error) {
	m := &Manager{tools: map[string]Tool{}}
	for _, name := range toolNames {
		t, err := MustNew(name)
		if err != nil {
			return nil, err
		}
		m.tools[name] = t
	}
	return m, nil
}

// Get 返回指定名称工具。
func (m *Manager) Get(name string) (Tool, error) {
	t, ok := m.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not enabled: %s", name)
	}
	return t, nil
}
