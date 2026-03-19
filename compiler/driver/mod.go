package driver

import (
	"encoding/json"
	"fmt"
	"os"
)

type ModConfig struct {
	Name string `json:"name"`
}

func loadModuleConfig(modulePath string) (*ModConfig, error) {
	// Check if the module config file exists.
	configPath := modulePath + "/au.mod"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("module config file not found: %s", configPath)
	}

	// Load the module config file.
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading module config file: %s", err)
	}

	var config ModConfig
	err = json.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing module config file: %s", err)
	}

	return &config, nil
}
