package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// repoConfig 仓库配置
type repoConfig struct {
	Repo           string  `json:"repo"`
	ChannelID      int64   `json:"channel_id,omitempty"`
	ChannelTitle   string  `json:"channel_title,omitempty"`
	MonitorRelease bool    `json:"monitor_releases"`
	MonitorCommit  bool    `json:"monitor_commits"`
	Branch         string  `json:"branch,omitempty"`
	LastReleaseID  *int64  `json:"last_release_id"`
	LastCommitSHA  *string `json:"last_commit_sha"`
}

var configMu sync.Mutex

// loadConfigs 加载配置文件
func loadConfigs() ([]repoConfig, error) {
	configMu.Lock()
	defer configMu.Unlock()

	_, err := os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return []repoConfig{}, nil
	}
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []repoConfig{}, nil
	}

	var configs []repoConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		log.Printf("ERROR: Failed to parse config file: %v", err)
		return nil, fmt.Errorf("corrupt config file, please check %s: %w", configFile, err)
	}
	return configs, nil
}

// saveConfigs 保存配置文件
func saveConfigs(configs []repoConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configs, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}
