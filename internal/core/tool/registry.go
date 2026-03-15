package tool

import (
	"fmt"
	"sync"
)

// Factory 用于按配置创建 Tool 实例。
type Factory func() Tool

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register 注册工具实现，建议在具体实现包的 init 中调用。
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// MustNew 根据名称创建工具实例。
func MustNew(name string) (Tool, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return f(), nil
}

// List 返回所有已注册工具名。
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
