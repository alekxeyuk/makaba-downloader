# 2ch Downloader

A Go application for downloading media files from specified threads on 2ch boards based on tags.

## Features

- Monitors multiple 2ch boards for new threads containing specific tags
- Downloads media files (webm, mp4, gif, jpg, png, etc.) from matching threads
- Concurrent downloads with configurable limits
- Graceful shutdown handling
- Configurable through JSON configuration
- Prevents duplicate downloads by tracking MD5 hashes
- Automatic directory creation for threads

## Configuration

The application uses `config.json` for configuration:

- `defaults`: Default settings including thread subject substrings, file extensions, and ignored substrings
- `boards`: List of 2ch boards to monitor with corresponding local directory names
- `tags`: List of tags to search for in threads
- `ignored_tags`: Tags to ignore even if they match
- `usercode_auth`: Authentication token for 2ch API

## Requirements

- Go 1.25.3 or higher
- `github.com/tidwall/gjson` v1.18.0

## Usage

1. Update `config.json` with your desired boards, tags, and authentication
2. Run the application: `go run main.go`
3. The application will continuously monitor the specified boards and download matching media files

## File Structure

Files are organized in directories based on the board, with subdirectories for each thread. Files are named using their MD5 hash plus the original filename to prevent conflicts.

## License

See COPYING for license information.