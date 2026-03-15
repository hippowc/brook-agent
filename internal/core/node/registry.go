package node

import (
	"fmt"
	"sync"

	"brook-agent/internal/core/memory"
	"brook-agent/internal/core/tool"
)

// BuildConfig 为节点构建阶段提供依赖。
type BuildConfig struct {
	Memory memory.Store
	Tools  *tool.Manager
}

// Factory 定义节点工厂函数签名。
type Factory func(cfg BuildConfig) Node

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register 注册节点实现，建议在实现包 init() 中执行。
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// MustNew 根据名称创建节点实例。
func MustNew(name string, cfg BuildConfig) (Node, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", name)
	}
	return f(cfg), nil
}
