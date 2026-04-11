// Package a2ui 提供与 A2UI（Agent-to-UI）兼容的 JSON Lines 流式消息封装，便于客户端渐进渲染。
// 规范参考：https://a2ui.org/ （此处实现常用子集，非全量 schema）。
package a2ui

// Envelope 单条 JSONL 消息的通用外壳。
type Envelope struct {
	Version string         `json:"v"`
	Kind    string         `json:"kind"`
	Payload map[string]any `json:"payload,omitempty"`
}
