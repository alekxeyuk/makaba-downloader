package main

import (
	"encoding/json"
	"os"
)

type Defaults struct {
	ThreadSubjSubstrings []string `json:"thread_subj_substrings"`
	FileExtensions       []string `json:"file_extensions"`
	IgnoredSubstrings    []string `json:"ignored_substrings"`
}

type BoardConfig struct {
	Board                string   `json:"board"`
	DirName              string   `json:"dir_name"`
	ThreadSubjSubstrings []string `json:"thread_subj_substrings,omitempty"`
	FileExtensions       []string `json:"file_extensions,omitempty"`
	IgnoredSubstrings    []string `json:"ignored_substrings,omitempty"`
}

type AppConfig struct {
	Defaults     Defaults      `json:"defaults"`
	Boards       []BoardConfig `json:"boards"`
	Tags         []string      `json:"tags"`
	IgnoredTags  []string      `json:"ignored_tags"`
	UsercodeAuth string        `json:"usercode_auth"`
}

func LoadConfig(filename string) (*AppConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config AppConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	// Apply defaults to boards that don't have custom settings
	for i := range config.Boards {
		if len(config.Boards[i].ThreadSubjSubstrings) == 0 {
			config.Boards[i].ThreadSubjSubstrings = config.Defaults.ThreadSubjSubstrings
		}
		if len(config.Boards[i].FileExtensions) == 0 {
			config.Boards[i].FileExtensions = config.Defaults.FileExtensions
		}
		if len(config.Boards[i].IgnoredSubstrings) == 0 {
			config.Boards[i].IgnoredSubstrings = config.Defaults.IgnoredSubstrings
		}
	}

	return &config, nil
}
