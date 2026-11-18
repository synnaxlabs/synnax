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
	"go.uber.org/zap/zapcore"
)

const StartedMessage = "started successfully"

// isCrashBanner reports whether line is the first line of a crash dump emitted by the
// driver's crash handler (x::crash). The dump is raw text rather than glog-formatted:
// a banner, an optional what() line, "stack trace:", and one line per frame.
func isCrashBanner(line string) bool {
	return strings.Contains(line, "*** ") &&
		(strings.Contains(line, " crashed: ") ||
			strings.Contains(line, " terminated: unhandled exception"))
}

// isFormatted reports whether line starts with the Driver's log prefix: a level byte,
// then a timestamp or a space.
func isFormatted(line string) bool {
	if len(line) < 2 || !strings.ContainsRune("DIWEF", rune(line[0])) {
		return false
	}
	return line[1] == ' ' || (line[1] >= '0' && line[1] <= '9')
}

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
func ParseLine(line, prevCaller string) ParsedLine {
	p := ParsedLine{Level: line[0], Caller: prevCaller}

	firstClose := strings.IndexByte(line, ']')
	if firstClose == -1 {
		p.Message = strings.TrimSpace(line)
		return p
	}

	secondClose := strings.IndexByte(line[firstClose+1:], ']')
	if secondClose == -1 {
		prefix := line[:firstClose]
		if _, caller, ok := strings.CutLast(prefix, " "); ok {
			p.Caller = caller
		}
		p.Message = strings.TrimSpace(line[firstClose+1:])
		return p
	}
	secondClose += firstClose + 1

	prefix := line[:firstClose]
	if _, caller, ok := strings.CutLast(prefix, " "); ok {
		p.Caller = strings.TrimPrefix(caller, "[")
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
//
// PipeToLogger calls onLoaded, if it is not nil, on the first formatted line or crash
// banner, with crashed set for a crash banner.
//
// PipeToLogger returns the last error the stream carried, or "" if there was none: an
// error or fatal line, the banner of a crash dump, or a line written before the first
// formatted line, such as a dynamic loader error.
func PipeToLogger(
	reader io.ReadCloser,
	logger *alamos.Logger,
	started chan<- struct{},
	startedOnce *sync.Once,
	onLoaded func(crashed bool),
) string {
	f := forwarder{
		logger:       logger,
		driverLogger: logger.WithOptions(zap.AddStacktrace(zapcore.FatalLevel)),
		loggers:      make(map[string]*alamos.Logger),
		started:      started,
		startedOnce:  startedOnce,
		onLoaded:     onLoaded,
	}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		f.forward(scanner.Bytes())
	}
	if len(f.crashLines) > 0 {
		f.driverLogger.Error(strings.Join(f.crashLines, "\n"))
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
	return f.lastErr
}

// forwarder holds the state of one PipeToLogger stream.
type forwarder struct {
	// logger logs errors of the reader, where a Go stacktrace is useful.
	logger *alamos.Logger
	// driverLogger logs the lines of the driver with no Go stacktrace: on a line of a
	// subprocess, it only points at the reader goroutine.
	driverLogger *alamos.Logger
	loggers      map[string]*alamos.Logger
	started      chan<- struct{}
	startedOnce  *sync.Once
	onLoaded     func(crashed bool)
	caller       string
	callerBuf    []byte
	lastErr      string
	formatted    bool
	// crashLines buffers a crash dump from its banner on, to log as one Error entry
	// when the stream closes. It is nil until the banner appears.
	crashLines []string
}

// forward handles one line of the stream. b is valid only until the next scan.
func (f *forwarder) forward(b []byte) {
	if len(b) == 0 {
		if f.crashLines == nil {
			f.logger.Warn("received empty log line from driver")
		}
		return
	}
	line := unsafe.String(unsafe.SliceData(b), len(b))
	if f.crashLines == nil && isCrashBanner(line) {
		if !f.formatted && f.onLoaded != nil {
			f.onLoaded(true)
		}
		f.crashLines = make([]string, 0, 32)
		f.lastErr = strings.Clone(strings.TrimSpace(line))
	}
	if f.crashLines != nil {
		// strings.Clone is required: the scanner reuses its buffer across
		// iterations, so line must be copied to survive until the flush.
		f.crashLines = append(f.crashLines, strings.Clone(line))
		return
	}
	p := ParseLine(line, f.caller)
	if isFormatted(line) {
		if !f.formatted && f.onLoaded != nil {
			f.onLoaded(false)
		}
		f.formatted = true
	} else if !f.formatted {
		f.lastErr = strings.Clone(strings.TrimSpace(line))
	}
	if p.Caller != f.caller {
		f.callerBuf = append(f.callerBuf[:0], p.Caller...)
		f.caller = unsafe.String(unsafe.SliceData(f.callerBuf), len(f.callerBuf))
	}
	if f.startedOnce != nil && p.Message == StartedMessage {
		f.startedOnce.Do(func() { close(f.started) })
	}
	f.log(p)
}

// log writes p to the logger of its name.
func (f *forwarder) log(p ParsedLine) {
	namedLogger := f.named(p.Name)
	callerField := zap.String("caller", f.caller)
	switch p.Level {
	case 'D':
		namedLogger.Debug(p.Message, callerField)
	case 'E', 'F':
		namedLogger.Error(p.Message, callerField)
		f.lastErr = strings.Clone(p.Message)
	case 'W':
		namedLogger.Warn(p.Message, callerField)
	default:
		namedLogger.Info(p.Message, callerField)
	}
}

// named returns the driver logger for name, and caches it.
func (f *forwarder) named(name string) *alamos.Logger {
	if name == "" {
		return f.driverLogger
	}
	if cached, ok := f.loggers[name]; ok {
		return cached
	}
	name = strings.Clone(name)
	l := f.driverLogger.Named(name)
	f.loggers[name] = l
	return l
}
