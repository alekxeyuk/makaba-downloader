package main

import (
	"context"
	"os"
	"time"
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
	setupGracefulShutdown(cancel)

	// Main processing loop
	for {
		if checkContextCancellation(ctx, downloader) {
			return
		}

		// Process all boards
		processAllBoards(ctx, api, downloader, appConfig, lastHits)

		// Wait for all downloads to complete
		downloader.Wait()

		// Save updated last hits
		saveLastHits(lastHits)

		// Sleep before next iteration
		if !sleepOrCancel(ctx, downloader, 180*time.Second) {
			return
		}
	}
}

// processAllBoards processes all configured boards
func processAllBoards(ctx context.Context, api *DvachApi, downloader *Downloader, appConfig *AppConfig, lastHits map[string]int64) {
	for _, conf := range appConfig.Boards {
		err := processBoard(ctx, api, downloader, conf, appConfig, lastHits)
		if err != nil {
			continue
		}
	}
}

// sleepOrCancel sleeps for the specified duration or returns early if context is cancelled
func sleepOrCancel(ctx context.Context, downloader *Downloader, duration time.Duration) bool {
	Log.Info("Done... Sleeping for %v", duration)

	select {
	case <-ctx.Done():
		Log.Info("Shutting down...")
		downloader.Stop()
		return false
	case <-time.After(duration):
		return true
	}
}
