package main

import (
	"strings"

	"github.com/tidwall/gjson"
)

type ThreadInfo struct {
	Data    []byte
	LastHit int64
}

func getThreads(api *DvachApi, catalog []byte, threadSubjSubstrings []string, ignoredSubstrings []string, logger *Logger, lastHits map[string]int64) []ThreadInfo {
	var threads []ThreadInfo
	boardID := gjson.GetBytes(catalog, "board.id").String()

	for _, thread := range gjson.GetBytes(catalog, "threads").Array() {
		comment := gjson.GetBytes([]byte(thread.Raw), "comment").String()
		tags := gjson.GetBytes([]byte(thread.Raw), "tags").String()
		subject := gjson.GetBytes([]byte(thread.Raw), "subject").String()

		// Check if thread contains desired substrings
		hasDesired := containsSubstring(comment, threadSubjSubstrings) ||
			containsSubstring(tags, threadSubjSubstrings) ||
			containsSubstring(subject, threadSubjSubstrings)

		// Check if thread contains ignored substrings
		hasIgnored := containsSubstring(comment, ignoredSubstrings) ||
			containsSubstring(tags, ignoredSubstrings) ||
			containsSubstring(subject, ignoredSubstrings)

		if hasDesired && !hasIgnored {
			threadNum := gjson.GetBytes([]byte(thread.Raw), "num").String()
			currentLastHit := gjson.GetBytes([]byte(thread.Raw), "files_count").Int()

			// Check if lasthit is newer than what we remember
			key := boardID + "_" + threadNum
			storedLastHit, exists := lastHits[key]
			if !exists || currentLastHit > storedLastHit {
				logger.Debug("Found matching thread with new activity: %s (lasthit: %d)", threadNum, currentLastHit)

				threadData, err := api.threadGet(boardID, threadNum)
				if err != nil {
					logger.Error("Error getting thread %s: %v", threadNum, err)
					continue
				}
				threads = append(threads, ThreadInfo{Data: threadData, LastHit: currentLastHit})
			}
		}
	}
	return threads
}

func containsSubstring(s string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substring)) {
			return true
		}
	}
	return false
}
