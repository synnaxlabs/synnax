// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

const StartedMessage = "started successfully"

// ParsedLine holds the parsed components of a single driver log line.
type ParsedLine struct {
	Level   byte
	Caller  string
	Name    string
	Message string
}

// ParseLine extracts the level, caller, logger name, and message from a raw
// driver log line. prevCaller is returned as Caller for continuation lines
// that lack their own caller information. The returned strings are substrings
// of line and share its backing memory.
func ParseLine(line string, prevCaller string) ParsedLine {
	p := ParsedLine{Level: line[0], Caller: prevCaller}

	firstClose := strings.IndexByte(line, ']')
	if firstClose == -1 {
		p.Message = strings.TrimSpace(line)
		return p
	}

	secondClose := strings.IndexByte(line[firstClose+1:], ']')
	if secondClose == -1 {
		prefix := line[:firstClose]
		if lastSpace := strings.LastIndexByte(prefix, ' '); lastSpace >= 0 {
			p.Caller = prefix[lastSpace+1:]
		}
		p.Message = strings.TrimSpace(line[firstClose+1:])
		return p
	}
	secondClose += firstClose + 1

	prefix := line[:firstClose]
	if lastSpace := strings.LastIndexByte(prefix, ' '); lastSpace >= 0 {
		p.Caller = strings.TrimPrefix(prefix[lastSpace+1:], "[")
	}
	p.Name = strings.TrimPrefix(strings.TrimSpace(line[firstClose+1:secondClose]), "[")
	msg := line[secondClose+1:]
	if len(msg) > 1 {
		msg = msg[1:]
	}
	p.Message = strings.TrimSpace(msg)
	return p
}

// PipeToLogger reads lines from reader and logs them using the provided logger.
// Each line is expected in the format "L [module] [caller] message" where L is a
// log level character (D/I/W/E/F). When a line containing StartedMessage is
// encountered, startedOnce is used to safely close the started channel exactly
// once, even when multiple goroutines call PipeToLogger concurrently.
func PipeToLogger(
	reader io.ReadCloser,
	logger *alamos.Logger,
	started chan<- struct{},
	startedOnce *sync.Once,
) {
	var (
		caller    string
		callerBuf []byte
	)
	loggers := make(map[string]*alamos.Logger)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			logger.Warn("received empty log line from driver")
			continue
		}
		line := unsafe.String(unsafe.SliceData(b), len(b))
		p := ParseLine(line, caller)

		if p.Caller != caller {
			callerBuf = append(callerBuf[:0], p.Caller...)
			caller = unsafe.String(unsafe.SliceData(callerBuf), len(callerBuf))
		}

		namedLogger := logger
		if p.Name != "" {
			if cached, ok := loggers[p.Name]; ok {
				namedLogger = cached
			} else {
				name := strings.Clone(p.Name)
				namedLogger = logger.Named(name)
				loggers[name] = namedLogger
			}
		}

		if startedOnce != nil && p.Message == StartedMessage {
			startedOnce.Do(func() { close(started) })
		}
		callerField := zap.String("caller", caller)
		switch p.Level {
		case 'D':
			namedLogger.Debug(p.Message, callerField)
		case 'E', 'F':
			namedLogger.Error(p.Message, callerField)
		case 'W':
			namedLogger.Warn(p.Message, callerField)
		default:
			namedLogger.Info(p.Message, callerField)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
}
