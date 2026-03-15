package common

import (
	"context"
	"log"
	"time"
)

// Event 是通用埋点事件定义，可覆盖入口、节点执行、工具调用等关键流程。
type Event struct {
	TraceID   string            `json:"trace_id,omitempty"`
	Name      string            `json:"name"`
	Timestamp time.Time         `json:"timestamp"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// Emitter 定义埋点事件发送器接口。
type Emitter interface {
	Emit(ctx context.Context, event Event) error
}

// CompositeEmitter 支持一处插桩、多处输出（日志、流式、外部采集系统）。
type CompositeEmitter struct {
	emitters []Emitter
}

// NewCompositeEmitter 创建组合发送器。
func NewCompositeEmitter(emitters ...Emitter) *CompositeEmitter {
	return &CompositeEmitter{emitters: emitters}
}

// Emit 依次发送埋点事件，单个发送器失败不影响其他发送器。
func (c *CompositeEmitter) Emit(ctx context.Context, event Event) error {
	for _, e := range c.emitters {
		if e == nil {
			continue
		}
		_ = e.Emit(ctx, event)
	}
	return nil
}

// LogEmitter 将埋点输出到标准日志。
type LogEmitter struct{}

func (LogEmitter) Emit(_ context.Context, event Event) error {
	log.Printf("[event] trace=%s name=%s fields=%v", event.TraceID, event.Name, event.Fields)
	return nil
}

// StreamEmitter 将埋点事件以流式 chunk 的形式发给前端。
type StreamEmitter struct {
	Writer StreamWriter
}

func (s StreamEmitter) Emit(ctx context.Context, event Event) error {
	if s.Writer == nil {
		return nil
	}
	return s.Writer.WriteChunk(ctx, StreamChunk{
		Type: "event",
		Data: event.Name,
	})
}
