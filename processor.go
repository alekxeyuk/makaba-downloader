package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tidwall/gjson"
)

// setupGracefulShutdown sets up signal handling for graceful shutdown
func setupGracefulShutdown(cancel context.CancelFunc) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		Log.Info("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()
}

// checkContextCancellation checks if context is cancelled and performs shutdown
func checkContextCancellation(ctx context.Context, downloader *Downloader) bool {
	select {
	case <-ctx.Done():
		Log.Info("Shutting down...")
		downloader.Stop()
		return true
	default:
		return false
	}
}

// processBoard processes a single board configuration
func processBoard(ctx context.Context, api *DvachApi, downloader *Downloader, conf BoardConfig, appConfig *AppConfig, lastHits map[string]int64) error {
	if checkContextCancellation(ctx, downloader) {
		return context.Canceled
	}

	catalog, err := api.catalogGet(conf.Board)
	if err != nil {
		Log.Error("Error getting catalog for %s: %v", conf.Board, err)
		return err
	}

	alreadyHaveFiles := getAlreadyHaveFiles(conf.DirName)

	threads, count := getThreads(api, catalog, appConfig.Tags, appConfig.IgnoredTags, lastHits)
	if len(threads) == 0 {
		Log.Warning("%s - No interesting threads found out of %d", conf.DirName, count)
		return nil
	}

	boardID := gjson.GetBytes(catalog, "board.id").String()

	for _, threadInfo := range threads {
		if checkContextCancellation(ctx, downloader) {
			return context.Canceled
		}

		err := processThread(ctx, api, downloader, conf, threadInfo, boardID, alreadyHaveFiles, lastHits)
		if err != nil {
			continue
		}
	}

	return nil
}

// processThread processes a single thread and downloads its files
func processThread(ctx context.Context, api *DvachApi, downloader *Downloader, conf BoardConfig, threadInfo ThreadInfo, boardID string, alreadyHaveFiles map[string]struct{}, lastHits map[string]int64) error {
	bigThreadNum := gjson.GetBytes(threadInfo.Data, "current_thread").String()
	threadDir := filepath.Join(conf.DirName, bigThreadNum)

	err := os.MkdirAll(threadDir, 0755)
	if err != nil {
		Log.Error("Error creating directory %s: %v", threadDir, err)
		return err
	}

	processThreadFiles(ctx, api, downloader, conf, threadInfo, threadDir, alreadyHaveFiles)

	// Update last hit for this thread
	key := boardID + "_" + bigThreadNum
	lastHits[key] = threadInfo.LastHit

	return nil
}

// processThreadFiles processes all files in a thread
func processThreadFiles(ctx context.Context, api *DvachApi, downloader *Downloader, conf BoardConfig, threadInfo ThreadInfo, threadDir string, alreadyHaveFiles map[string]struct{}) {
	for _, thread := range gjson.GetBytes(threadInfo.Data, "threads").Array() {
		for _, post := range gjson.GetBytes([]byte(thread.Raw), "posts").Array() {
			files := gjson.GetBytes([]byte(post.Raw), "files").Array()
			for _, postFile := range files {
				if checkContextCancellation(ctx, downloader) {
					return
				}

				processFile(api, downloader, conf, postFile, threadDir, alreadyHaveFiles)
			}
		}
	}
}

// processFile processes a single file from a post
func processFile(api *DvachApi, downloader *Downloader, conf BoardConfig, postFile gjson.Result, threadDir string, alreadyHaveFiles map[string]struct{}) {
	md5 := gjson.GetBytes([]byte(postFile.Raw), "md5").String()

	// Check if we already have this file
	if _, ok := alreadyHaveFiles[md5]; ok {
		return
	}

	// Check if file extension is valid
	if !isValidFileExtension([]byte(postFile.Raw), conf.FileExtensions) {
		Log.Info("Unknown file format: %s", gjson.GetBytes([]byte(postFile.Raw), "path").String())
		return
	}

	fileURL := api.url + gjson.GetBytes([]byte(postFile.Raw), "path").String()

	// Skip stickers
	if strings.Contains(fileURL, "stickers") {
		return
	}

	// Generate filename
	fileName := generateFileName(postFile, threadDir, md5)
	fileName = sanitizeFileName(fileName)

	downloader.DownloadFileAsync(fileURL, fileName)
	alreadyHaveFiles[md5] = struct{}{}
}

// generateFileName generates a filename for a file
func generateFileName(postFile gjson.Result, threadDir string, md5 string) string {
	fullname := gjson.GetBytes([]byte(postFile.Raw), "fullname").String()

	// Truncate fullname if too long
	if len(fullname) > 128 {
		fullname = fullname[len(fullname)-32:]
	}

	var fileName string
	if strings.Contains(fullname, ".") {
		fileName = filepath.Join(threadDir, fmt.Sprintf("%s_%s", md5, fullname))
	} else {
		path := gjson.GetBytes([]byte(postFile.Raw), "path").String()
		ext := filepath.Ext(path)
		fileName = filepath.Join(threadDir, fmt.Sprintf("%s_%s%s", md5, fullname, ext))
	}

	return fileName
}
