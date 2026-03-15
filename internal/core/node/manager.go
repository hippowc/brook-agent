package node

import "fmt"

// Manager 统一管理已启用节点。
type Manager struct {
	nodes map[string]Node
}

// NewManager 按节点名称清单构建管理器。
func NewManager(names []string, cfg BuildConfig) (*Manager, error) {
	m := &Manager{nodes: map[string]Node{}}
	for _, name := range names {
		n, err := MustNew(name, cfg)
		if err != nil {
			return nil, err
		}
		m.nodes[name] = n
	}
	return m, nil
}

// Get 返回指定节点实例。
func (m *Manager) Get(name string) (Node, error) {
	n, ok := m.nodes[name]
	if !ok {
		return nil, fmt.Errorf("node not enabled: %s", name)
	}
	return n, nil
}
