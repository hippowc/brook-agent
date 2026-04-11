package store

import (
	"encoding/json"
	"os"
)

// SessionFile 将业务层 session（map）持久化到单个 JSON 文件，用于跨进程恢复「会话变量」草稿。
type SessionFile struct {
	Path string
}

// Load 读取 JSON 对象到 map；文件不存在时返回空 map。
func (s *SessionFile) Load() (map[string]any, error) {
	if s.Path == "" {
		return map[string]any{}, nil
	}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// Save 将 map 写回文件。
func (s *SessionFile) Save(data map[string]any) error {
	if s.Path == "" {
		return nil
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
