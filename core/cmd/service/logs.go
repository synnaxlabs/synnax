// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux || windows

package service

import (
	"bufio"
	"encoding/json/v2"
	"os"
	"time"

	"github.com/synnaxlabs/x/errors"
)

// logEntry represents a parsed JSON log entry from zap.
type logEntry struct {
	Level  string  `json:"level"`
	TS     float64 `json:"ts"`
	Logger string  `json:"logger"`
	Caller string  `json:"caller"`
	Msg    string  `json:"msg"`
	Error  string  `json:"error"`
}

// ANSI color codes for log levels.
const (
	colorReset   = "\033[0m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorBoldRed = "\033[1;31m"
)

// logLevelInfo contains display information for each log level.
type logLevelInfo struct {
	color string
	label string
}

// importantLogLevels maps log levels to their display info.
var importantLogLevels = map[string]logLevelInfo{
	"warn":   {color: colorYellow, label: "WARN"},
	"error":  {color: colorRed, label: "ERROR"},
	"fatal":  {color: colorBoldRed, label: "FATAL"},
	"panic":  {color: colorBoldRed, label: "PANIC"},
	"dpanic": {color: colorBoldRed, label: "DPANIC"},
}

// readRecentLogs reads the log file and returns formatted important log entries.
func readRecentLogs(filePath string, maxEntries int) (_ []string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(f.Close(), "failed to close log file"))
	}()

	var entries []logEntry
	scanner := bufio.NewScanner(f)
	// Increase buffer size for long log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		var entry logEntry
		if jsonErr := json.Unmarshal([]byte(line), &entry); jsonErr != nil {
			continue // Skip non-JSON lines
		}
		if _, ok := importantLogLevels[entry.Level]; ok {
			entries = append(entries, entry)
			if len(entries) > maxEntries {
				entries = entries[1:]
			}
		}
	}

	// Format entries for display
	var lines []string
	for _, e := range entries {
		ts := time.Unix(int64(e.TS), int64((e.TS-float64(int64(e.TS)))*1e9))
		info := importantLogLevels[e.Level]
		msg := e.Msg
		if e.Error != "" {
			msg += ": " + e.Error
		}
		line := ts.Format(
			"2006-01-02 15:04:05",
		) + " " + info.color + info.label + colorReset + " " + msg
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}
