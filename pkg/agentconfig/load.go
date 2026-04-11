package agentconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFile 从 YAML 文件加载 Root（并展开 instruction/user_prompt 中的 @文件路径）。
func LoadFile(path string) (*Root, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadYAMLWithDir(b, filepath.Dir(path))
}

// LoadYAMLWithDir 解析 YAML 并在给定配置目录下展开 @ 文件引用。
func LoadYAMLWithDir(b []byte, configDir string) (*Root, error) {
	var r Root
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if err := ExpandAtFileRefs(&r, configDir); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// LoadYAML 解析 YAML 字节（无配置目录时不展开 @ 引用）。
func LoadYAML(b []byte) (*Root, error) {
	var r Root
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}
