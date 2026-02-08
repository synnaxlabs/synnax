// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package log

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

const StartedMessage = "started successfully"

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
	var caller string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		b := scanner.Bytes()
		namedLogger := logger
		if len(b) == 0 {
			namedLogger.Warn("received empty log line from driver")
			continue
		}
		level := string(b[0])
		original := string(b)
		split := strings.Split(original, "]")
		message := original
		if len(split) >= 3 {
			callerSplit := strings.Split(split[0], " ")
			caller = strings.TrimPrefix(callerSplit[len(callerSplit)-1], "[")
			first := strings.TrimPrefix(strings.TrimSpace(split[1]), "[")
			namedLogger = logger.Named(first)
			message = split[2]
			if len(message) > 1 {
				message = message[1:]
			}
		} else if len(split) == 2 {
			callerSplit := strings.Split(split[0], " ")
			caller = callerSplit[len(callerSplit)-1]
			message = split[1]
		}
		message = strings.TrimSpace(message)
		if startedOnce != nil && message == StartedMessage {
			startedOnce.Do(func() { close(started) })
		}
		callerField := zap.String("caller", caller)
		switch level {
		case "D":
			namedLogger.Debug(message, callerField)
		case "E", "F":
			namedLogger.Error(message, callerField)
		case "W":
			namedLogger.Warn(message, callerField)
		default:
			namedLogger.Info(message, callerField)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
}
