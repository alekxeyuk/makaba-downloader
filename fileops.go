package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

func loadLastHits() map[string]int64 {
	lastHits := make(map[string]int64)
	file, err := os.Open("lasthits.json")
	if err != nil {
		if !os.IsNotExist(err) {
			Log.Error("Error opening lasthits.json: %v", err)
		}
		return lastHits
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&lastHits)
	if err != nil {
		Log.Error("Error decoding lasthits.json: %v", err)
	}
	return lastHits
}

func saveLastHits(lastHits map[string]int64) {
	file, err := os.Create("lasthits.json")
	if err != nil {
		Log.Error("Error creating lasthits.json: %v", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(lastHits)
	if err != nil {
		Log.Error("Error encoding lasthits.json: %v", err)
	}
}

func getAlreadyHaveFiles(dirName string) map[string]struct{} {
	alreadyHaveFiles := make(map[string]struct{})

	// Check if directory exists
	fi, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		Log.Info("Directory %s does not exist, skipping...", dirName)
		return alreadyHaveFiles
	}
	if err != nil {
		Log.Error("Error checking directory %s: %v", dirName, err)
		return alreadyHaveFiles
	}
	if !fi.IsDir() {
		Log.Error("Path %s is not a directory", dirName)
		return alreadyHaveFiles
	}

	// Use WalkDir for better performance (available in Go 1.16+)
	err = filepath.WalkDir(dirName, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			name := info.Name()
			if len(name) >= 32 {
				alreadyHaveFiles[name[:32]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		Log.Error("Error walking directory %s: %v", dirName, err)
	}
	return alreadyHaveFiles
}

func sanitizeFileName(fileName string) string {
	// Characters that are invalid in filenames on Windows and most Unix systems
	invalidChars := []string{":", "*", "?", "<", ">", "|", "\""}
	for _, char := range invalidChars {
		fileName = strings.ReplaceAll(fileName, char, "")
	}
	return fileName
}

func isValidFileExtension(file []byte, fileExtensions []string) bool {
	path := gjson.GetBytes(file, "path").String()
	ext := filepath.Ext(path)
	if len(ext) == 0 {
		return false
	}
	for _, allowedExt := range fileExtensions {
		if strings.EqualFold(ext[1:], allowedExt) {
			return true
		}
	}
	// Log unknown extension to unknown.txt
	unknownFile, err := os.OpenFile("unknown.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer unknownFile.Close()
		unknownFile.WriteString(ext[1:] + "\n")
	}
	return false
}
