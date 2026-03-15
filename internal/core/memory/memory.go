package memory

import (
	"context"
	"time"

	"brook-agent/internal/model"
)

// Session 保存一次会话期间的所有上下文信息。
type Session struct {
	ID          string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Messages    []model.Message
	ToolResults []model.ToolResult
	Variables   map[string]string
}

// Store 定义记忆系统接口，支持会话生命周期与过程数据读写。
type Store interface {
	GetOrCreate(ctx context.Context, sessionID string) (*Session, error)
	SaveMessage(ctx context.Context, sessionID string, msg model.Message) error
	SaveToolResult(ctx context.Context, sessionID string, result model.ToolResult) error
	UpdateVariables(ctx context.Context, sessionID string, vars map[string]string) error
}
