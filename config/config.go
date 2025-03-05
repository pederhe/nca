package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config structure
type Config map[string]string

// Get local config file path
func getLocalConfigPath() string {
	return filepath.Join(".nca", "config")
}

// Get global config file path
func getGlobalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".nca_config")
}

// Load configuration
func loadConfig(isGlobal bool) Config {
	var path string
	if isGlobal {
		path = getGlobalConfigPath()
	} else {
		path = getLocalConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}
	}

	return config
}

// Save configuration
func saveConfig(config Config, isGlobal bool) error {
	var path string
	if isGlobal {
		path = getGlobalConfigPath()
	} else {
		path = getLocalConfigPath()
		// Ensure directory exists
		os.MkdirAll(filepath.Dir(path), 0755)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get configuration value
func Get(key string) string {
	// Try to get from local config first
	localConfig := loadConfig(false)
	if value, ok := localConfig[key]; ok {
		return value
	}

	// If not in local, try global config
	globalConfig := loadConfig(true)
	return globalConfig[key]
}

// Set configuration value
func Set(key, value string, isGlobal bool) error {
	config := loadConfig(isGlobal)
	config[key] = value
	return saveConfig(config, isGlobal)
}

// Remove configuration value
func Unset(key string, isGlobal bool) error {
	config := loadConfig(isGlobal)
	delete(config, key)
	return saveConfig(config, isGlobal)
} 