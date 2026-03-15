package inmemory

import (
	"context"
	"sync"
	"time"

	"brook-agent/internal/core/memory"
	"brook-agent/internal/model"
)

// Store 是 memory.Store 的内存实现，适合本地开发和单进程运行。
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*memory.Session
}

// New 创建内存版记忆存储。
func New() *Store {
	return &Store{
		sessions: make(map[string]*memory.Session),
	}
}

// GetOrCreate 根据 sessionID 获取或创建会话。
func (s *Store) GetOrCreate(_ context.Context, sessionID string) (*memory.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.sessions[sessionID]; ok {
		return sess, nil
	}

	now := time.Now()
	sess := &memory.Session{
		ID:        sessionID,
		CreatedAt: now,
		UpdatedAt: now,
		Variables: map[string]string{},
	}
	s.sessions[sessionID] = sess
	return sess, nil
}

// SaveMessage 写入消息历史。
func (s *Store) SaveMessage(_ context.Context, sessionID string, msg model.Message) error {
	sess, err := s.GetOrCreate(context.Background(), sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Messages = append(sess.Messages, msg)
	sess.UpdatedAt = time.Now()
	return nil
}

// SaveToolResult 写入工具调用结果。
func (s *Store) SaveToolResult(_ context.Context, sessionID string, result model.ToolResult) error {
	sess, err := s.GetOrCreate(context.Background(), sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	sess.ToolResults = append(sess.ToolResults, result)
	sess.UpdatedAt = time.Now()
	return nil
}

// UpdateVariables 批量更新会话变量。
func (s *Store) UpdateVariables(_ context.Context, sessionID string, vars map[string]string) error {
	sess, err := s.GetOrCreate(context.Background(), sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range vars {
		sess.Variables[k] = v
	}
	sess.UpdatedAt = time.Now()
	return nil
}
