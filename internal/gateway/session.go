package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hippowc/brook/internal/business/store"
	"github.com/hippowc/brook/internal/brookdir"
	"github.com/hippowc/brook/pkg/agentconfig"
)

// SessionStore 按外部用户隔离的 ADK SessionValues（与 memory.output_key 等兼容）。
type SessionStore interface {
	Load(key string) (map[string]any, error)
	Save(key string, data map[string]any) error
}

type memoryStore struct {
	mu   sync.Mutex
	data map[string]map[string]any
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: make(map[string]map[string]any)}
}

func (m *memoryStore) Load(key string) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.data[key]; ok {
		cp := make(map[string]any, len(v))
		for k, val := range v {
			cp[k] = val
		}
		return cp, nil
	}
	return map[string]any{}, nil
}

func (m *memoryStore) Save(key string, data map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make(map[string]any, len(data))
	for k, v := range data {
		cp[k] = v
	}
	m.data[key] = cp
	return nil
}

type fileStore struct {
	dir string
}

func newFileStore(dir string) (*fileStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &fileStore{dir: dir}, nil
}

func (f *fileStore) pathFor(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(f.dir, hex.EncodeToString(h[:])+".json")
}

func (f *fileStore) Load(key string) (map[string]any, error) {
	sf := store.SessionFile{Path: f.pathFor(key)}
	return sf.Load()
}

func (f *fileStore) Save(key string, data map[string]any) error {
	sf := store.SessionFile{Path: f.pathFor(key)}
	return sf.Save(data)
}

// NewSessionStore 由配置构造会话存储。
func NewSessionStore(g *agentconfig.GatewaySpec) (SessionStore, error) {
	switch strings.ToLower(strings.TrimSpace(g.Session.Store)) {
	case "memory":
		return newMemoryStore(), nil
	case "file", "":
		dir := strings.TrimSpace(g.Session.FileDir)
		if dir == "" {
			var err error
			dir, err = brookdir.GatewaySessionsDir()
			if err != nil {
				return nil, err
			}
		}
		return newFileStore(dir)
	default:
		return nil, fmt.Errorf("gateway: unknown session.store %q", g.Session.Store)
	}
}

// SessionKey 由 user_id 与可选 conversation_id 派生稳定键。
func SessionKey(userID, conversationID string) string {
	userID = strings.TrimSpace(userID)
	conversationID = strings.TrimSpace(conversationID)
	raw := userID + "\n" + conversationID
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
