package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/schema"
)

// FileVersion 当前存档格式版本，便于以后迁移。
const FileVersion = 1

// File 为 ~/.brook/conversations/<uuid>.json 磁盘格式。
type File struct {
	Version    int       `json:"version"`
	ID         string    `json:"id"`
	UpdatedAt  time.Time `json:"updated_at"`
	ConfigPath string    `json:"config_path,omitempty"`
	// Messages 使用与 schema.Message 一致的 JSON 结构，便于与 ADK 互操作。
	Messages []schema.Message `json:"messages"`
}

// Load 读取对话文件。
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Version == 0 {
		f.Version = 1
	}
	if err := ValidateID(f.ID); err != nil {
		return nil, err
	}
	return &f, nil
}

// Save 原子写入对话文件（同目录临时文件再 rename）。
func Save(path string, f *File) error {
	if f == nil {
		return nil
	}
	if err := ValidateID(f.ID); err != nil {
		return err
	}
	f.Version = FileVersion
	f.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// MessagesPointers 供 adk.Runner.Run 使用。
func (f *File) MessagesPointers() []*schema.Message {
	out := make([]*schema.Message, len(f.Messages))
	for i := range f.Messages {
		mi := f.Messages[i]
		out[i] = &mi
	}
	return out
}

// SetFromMessages 从指针切片写回值切片以便 JSON 序列化。
func (f *File) SetFromMessages(msgs []*schema.Message) {
	f.Messages = make([]schema.Message, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		f.Messages = append(f.Messages, *m)
	}
}
