// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package progress logs the progress of a migration that visits many entries.
package progress

import (
	"github.com/synnaxlabs/alamos"
	"go.uber.org/zap"
)

const interval = 1000

// Logger logs the number of entries a migration has visited at logarithmic intervals
// (1, 10, 100, 1000), then every 1000 entries.
type Logger struct {
	// Instrumentation is the instrumentation the Logger logs through.
	alamos.Instrumentation
	// Key identifies the migration the Logger reports on.
	Key   string
	count int
}

// Increment counts one more visited entry and logs at the progress interval.
func (l *Logger) Increment() {
	l.count++
	if l.due() {
		l.L.Info(
			"migration progress",
			zap.String("migration", l.Key),
			zap.Int("entries", l.count),
		)
	}
}

func (l *Logger) due() bool {
	if l.count >= interval {
		return l.count%interval == 0
	}
	n := l.count
	for n >= 10 {
		if n%10 != 0 {
			return false
		}
		n /= 10
	}
	return n == 1
}
