package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const keyEnv = "ENV"
const envLocal = "local"

type Config struct {
	config *viper.Viper
}

func Load(env string) (*Config, error) {

	if len(env) == 0 {
		if env = os.Getenv(keyEnv); len(env) == 0 {
			env = envLocal
		}
	}

	configPath, err := getConfigPath(env)

	viperConfig := viper.New()
	if err == nil {
		viperConfig.SetConfigFile(configPath)
		if err := viperConfig.ReadInConfig(); err != nil {
			slog.Warn(fmt.Sprintf("error reading config file, %s", err))
		}
	}
	viperConfig.AutomaticEnv()

	cfg := &Config{
		config: viperConfig,
	}

	return cfg, nil
}

func (c *Config) GetPort() string {
	port := c.config.GetString("PORT")
	if len(port) == 0 {
		port = c.config.GetString("server.port")
	}

	return port
}

func (c *Config) GetKVDBPath() string {
	kvdbPath := c.config.GetString("KVDB_PATH")
	if len(kvdbPath) == 0 {
		kvdbPath = c.config.GetString("database.kvdb_path")
	}

	return kvdbPath
}

func (c *Config) GetIndexPath() string {
	indexPath := c.config.GetString("INDEX_PATH")
	if len(indexPath) == 0 {
		indexPath = c.config.GetString("database.index_path")
	}

	return indexPath
}

func (c *Config) GetStoragePath() string {
	storagePath := c.config.GetString("STORAGE_PATH")
	if len(storagePath) == 0 {
		storagePath = c.config.GetString("database.storage_path")
	}

	return storagePath
}

func getProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	for {
		configDir := filepath.Join(currentDir, "config")
		if info, err := os.Stat(configDir); err == nil && info.IsDir() {
			return currentDir, nil
		}

		parent := filepath.Dir(currentDir)

		if parent == currentDir {
			break
		}

		currentDir = parent
	}

	return "", fmt.Errorf("could not find project root (directory containing 'config' folder)")
}

func getConfigPath(env string) (string, error) {
	configFile := fmt.Sprintf("config.%s.yaml", env)

	projectRoot, err := getProjectRoot()
	if err != nil {
		slog.Warn("failed to find project root with config directory, will use environment variables instead", "err", err.Error())
		return "", fmt.Errorf("failed to find project root: %w", err)
	}
	configPath := filepath.Join(projectRoot, "config", configFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Warn("failed to find config file within config directory, will use environment variables instead", "err", err.Error())
		return "", fmt.Errorf("config file does not exist: %s", configPath)
	}

	return configPath, nil
}
