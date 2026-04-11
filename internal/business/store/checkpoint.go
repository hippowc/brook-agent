// Package store 提供基于文件的 CheckPointStore 实现，供 adk.Runner 使用。
package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// FileCheckPointStore 将 checkpoint 字节落盘，键为 checkpoint id 的 SHA256 文件名。
type FileCheckPointStore struct {
	Dir string
}

// NewFileCheckPointStore 在 dir 目录下创建存储；目录不存在时会尝试创建。
func NewFileCheckPointStore(dir string) (*FileCheckPointStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("checkpoint: empty dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileCheckPointStore{Dir: dir}, nil
}

func (f *FileCheckPointStore) path(id string) string {
	sum := sha256.Sum256([]byte(id))
	name := hex.EncodeToString(sum[:]) + ".chk"
	return filepath.Join(f.Dir, name)
}

// Get 实现 adk 使用的 CheckPointStore 语义。
func (f *FileCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	_ = ctx
	p := f.path(checkPointID)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}

// Set 写入或覆盖 checkpoint 文件。
func (f *FileCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	_ = ctx
	p := f.path(checkPointID)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, checkPoint, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
