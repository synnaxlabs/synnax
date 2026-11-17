// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"slices"

	"github.com/synnaxlabs/x/telem"
)

const (
	// errorLogInterval is the least time between two lines for one error of a node.
	errorLogInterval = 5 * telem.Second
	// maxLoggedErrors is the most different errors a node logs in one interval.
	maxLoggedErrors = 10
)

type loggedError struct {
	text string
	// at is the stamp of the cycle that logged the line.
	at telem.TimeStamp
}

// errorLog limits the log lines of one node. A node logs each different error one time
// in each errorLogInterval, and at most maxLoggedErrors different errors.
type errorLog struct {
	// logged holds the errors logged in the last errorLogInterval.
	logged []loggedError
	// skipped counts the errors not logged since the last line.
	skipped int
}

// take reports whether the node can log text in the cycle stamped now. With a line, it
// returns the count of errors not logged since the last line.
func (l *errorLog) take(text string, now telem.TimeStamp) (skipped int, ok bool) {
	l.logged = slices.DeleteFunc(l.logged, func(e loggedError) bool {
		return e.at.Span(now) >= errorLogInterval
	})
	logged := slices.ContainsFunc(l.logged, func(e loggedError) bool {
		return e.text == text
	})
	if logged || len(l.logged) >= maxLoggedErrors {
		l.skipped++
		return 0, false
	}
	l.logged = append(l.logged, loggedError{text: text, at: now})
	skipped, l.skipped = l.skipped, 0
	return skipped, true
}

// errorLogs holds the errorLog of each node of a runtime, by node key.
type errorLogs struct {
	// now is the stamp of the current cycle.
	now   telem.TimeStamp
	nodes map[string]*errorLog
}
