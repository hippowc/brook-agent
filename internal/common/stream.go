package common

import "context"

// StreamChunk 表示流式返回的最小数据单元。
type StreamChunk struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// StreamWriter 定义统一流式输出接口，entry 可按自身协议实现 SSE/CLI 打印等。
type StreamWriter interface {
	WriteChunk(ctx context.Context, chunk StreamChunk) error
	Close(ctx context.Context) error
}

// NopStreamWriter 用于不需要流式输出时的空实现。
type NopStreamWriter struct{}

func (NopStreamWriter) WriteChunk(context.Context, StreamChunk) error { return nil }
func (NopStreamWriter) Close(context.Context) error                   { return nil }
