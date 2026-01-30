package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tidwall/gjson"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appConfig, err := LoadConfig("config.json")
	if err != nil {
		Log.Error("Error loading config: %v", err)
		os.Exit(1)
	}

	api := NewDvachApi(map[string]string{
		"usercode_auth": appConfig.UsercodeAuth,
		"ageallow":      "1",
	})

	downloader := NewDownloader(api.client, 5) // Max 5 concurrent downloads

	// Load last hits from file
	lastHits := loadLastHits()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		Log.Info("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			Log.Info("Shutting down...")
			downloader.Stop()
			return
		default:
		}

		for _, conf := range appConfig.Boards {
			select {
			case <-ctx.Done():
				Log.Info("Shutting down...")
				downloader.Stop()
				return
			default:
			}

			catalog, err := api.catalogGet(conf.Board)
			if err != nil {
				Log.Error("Error getting catalog for %s: %v", conf.Board, err)
				continue
			}

			alreadyHaveFiles := getAlreadyHaveFiles(conf.DirName)

			threads := getThreads(api, catalog, appConfig.Tags, appConfig.IgnoredTags, lastHits)
			if len(threads) == 0 {
				Log.Warning("No %s threads found", conf.DirName)
				continue
			}

			for _, threadInfo := range threads {
				select {
				case <-ctx.Done():
					Log.Info("Shutting down...")
					downloader.Stop()
					return
				default:
				}

				bigThreadNum := gjson.GetBytes(threadInfo.Data, "current_thread").String()
				err := os.MkdirAll(filepath.Join(conf.DirName, bigThreadNum), 0755)
				if err != nil {
					Log.Error("Error creating directory %s: %v", filepath.Join(conf.DirName, bigThreadNum), err)
					continue
				}

				for _, thread := range gjson.GetBytes(threadInfo.Data, "threads").Array() {
					for _, post := range gjson.GetBytes([]byte(thread.Raw), "posts").Array() {
						files := gjson.GetBytes([]byte(post.Raw), "files").Array()
						for _, postFile := range files {
							md5 := gjson.GetBytes([]byte(postFile.Raw), "md5").String()
							if _, ok := alreadyHaveFiles[md5]; !ok {
								if isValidFileExtension([]byte(postFile.Raw), conf.FileExtensions) {
									fileURL := api.url + gjson.GetBytes([]byte(postFile.Raw), "path").String()

									if strings.Contains(fileURL, "stickers") {
										continue
									}

									fullname := gjson.GetBytes([]byte(postFile.Raw), "fullname").String()
									if len(fullname) > 128 {
										fullname = fullname[len(fullname)-32:]
									}

									var fileName string
									if strings.Contains(fullname, ".") {
										fileName = filepath.Join(conf.DirName, bigThreadNum, fmt.Sprintf("%s_%s", md5, fullname))
									} else {
										path := gjson.GetBytes([]byte(postFile.Raw), "path").String()
										ext := filepath.Ext(path)
										fileName = filepath.Join(conf.DirName, bigThreadNum, fmt.Sprintf("%s_%s%s", md5, fullname, ext))
									}

									fileName = sanitizeFileName(fileName)

									downloader.DownloadFileAsync(fileURL, fileName)
									alreadyHaveFiles[md5] = struct{}{}
								} else {
									Log.Info("Unknown file format: %s", gjson.GetBytes([]byte(postFile.Raw), "path").String())
								}
							}
						}
					}
				}

				// Update last hit for this thread
				boardID := gjson.GetBytes(catalog, "board.id").String()
				key := boardID + "_" + bigThreadNum
				lastHits[key] = threadInfo.LastHit
			}
		}

		downloader.Wait()

		// Save updated last hits
		saveLastHits(lastHits)

		Log.Info("Done... Sleeping for 180 sec")

		select {
		case <-ctx.Done():
			Log.Info("Shutting down...")
			downloader.Stop()
			return
		case <-time.After(180 * time.Second):
		}
	}
}
