package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const indexFile = "_index.json"
const maxIndexEntries = 64

// IndexEntry 用于在 conversations 目录下列出最近会话（可选）。
type IndexEntry struct {
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	Preview   string    `json:"preview,omitempty"`
	Config    string    `json:"config_path,omitempty"`
}

// Index 为 ~/.brook/conversations/_index.json。
type Index struct {
	Conversations []IndexEntry `json:"conversations"`
}

// UpdateIndex 在保存某会话后刷新索引（按时间倒序，截断条数）。
func UpdateIndex(convDir, id, configPath, preview string) error {
	if convDir == "" || id == "" {
		return nil
	}
	path := filepath.Join(convDir, indexFile)
	var idx Index
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &idx)
	}
	now := time.Now().UTC()
	found := -1
	for i := range idx.Conversations {
		if idx.Conversations[i].ID == id {
			found = i
			break
		}
	}
	ent := IndexEntry{ID: id, UpdatedAt: now, Preview: preview, Config: configPath}
	if found >= 0 {
		idx.Conversations[found] = ent
	} else {
		idx.Conversations = append(idx.Conversations, ent)
	}
	sort.Slice(idx.Conversations, func(i, j int) bool {
		return idx.Conversations[i].UpdatedAt.After(idx.Conversations[j].UpdatedAt)
	})
	if len(idx.Conversations) > maxIndexEntries {
		idx.Conversations = idx.Conversations[:maxIndexEntries]
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
