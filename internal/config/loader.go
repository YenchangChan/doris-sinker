package config

import (
	"os"

	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeConfigLoad, "failed to read config file", err)
	}

	// 解析YAML
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, errors.Wrap(errors.ErrCodeConfigLoad, "failed to parse config file", err)
	}

	// 从环境变量覆盖敏感信息
	if password := os.Getenv("DORIS_PASSWORD"); password != "" {
		cfg.Doris.Password = password
	}

	// 验证配置
	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
