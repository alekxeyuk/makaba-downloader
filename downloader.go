package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type downloaderError struct {
	err    error
	ignore bool
}

func (e *downloaderError) Error() string {
	return fmt.Sprintf("error downloading %s, ignore %v", e.err, e.ignore)
}

type Downloader struct {
	client  *http.Client
	limiter *rate.Limiter
	sem     chan struct{}
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewDownloader(client *http.Client, maxConcurrent int) *Downloader {
	ctx, cancel := context.WithCancel(context.Background())
	return &Downloader{
		client:  client,
		limiter: rate.NewLimiter(rate.Every(200*time.Millisecond), 10), // 10 requests per second
		sem:     make(chan struct{}, maxConcurrent),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (d *Downloader) DownloadFile(url, filepath string) error {
	select {
	case <-d.ctx.Done():
		return d.ctx.Err()
	default:
	}

	d.sem <- struct{}{}
	defer func() { <-d.sem }()

	err := d.limiter.Wait(d.ctx)
	if err != nil {
		return err
	}

	log.Printf("\033[32mDownloading %s\033[0m", url)

	tempFile := filepath + ".tmp"

	const maxRetries = 10
	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("Retrying download (attempt %d/%d) after %v", attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		err := d.downloadWithResume(url, tempFile)
		if err == nil {
			// Success - rename temp file to final destination
			if err := os.Rename(tempFile, filepath); err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("error renaming temp file: %w", err)
			}
			return nil
		} else if e, ok := err.(*downloaderError); ok && e.ignore {
			return err
		}

		// Check if it's a context cancellation - don't retry
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			os.Remove(tempFile)
			return err
		}

		log.Printf("Download attempt %d failed: %v", attempt+1, err)
	}

	os.Remove(tempFile)
	return fmt.Errorf("failed to download after %d attempts", maxRetries)
}

func (d *Downloader) downloadWithResume(url, filepath string) error {
	// Check if partial file exists
	var bytesDownloaded int64
	fileInfo, err := os.Stat(filepath)
	if err == nil {
		bytesDownloaded = fileInfo.Size()
	}

	// Create or open file for appending
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create request with Range header if resuming
	req, err := http.NewRequestWithContext(d.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	if bytesDownloaded > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesDownloaded))
		log.Printf("Resuming download %s from byte %d", filepath, bytesDownloaded)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	// 200 = full content, 206 = partial content (resume), both are OK
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent || resp.StatusCode == http.StatusNotFound {
		return &downloaderError{
			err:    fmt.Errorf("unexpected status code: %d", resp.StatusCode),
			ignore: true,
		}
	}

	log.Printf("\033[33mSaving file to %s\033[0m", filepath)

	// Copy with a wrapper that can detect context cancellation
	_, err = d.copyWithContext(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error copying data: %w", err)
	}

	return nil
}

func (d *Downloader) copyWithContext(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		select {
		case <-d.ctx.Done():
			return written, d.ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return written, er
			}
			break
		}
	}
	return written, nil
}

func (d *Downloader) DownloadFileAsync(url, filepath string) {
	d.wg.Go(func() {
		if err := d.DownloadFile(url, filepath); err != nil {
			log.Printf("\033[31mError downloading %s: %v\033[0m", url, err)
		}
	})
}

func (d *Downloader) Wait() {
	d.wg.Wait()
}

func (d *Downloader) Stop() {
	d.cancel()
	d.Wait()
}
