// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"fmt"
	"strconv"
	"time"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Warning is the text and data of a task warning.
type Warning struct {
	// Message says what happened in one short line.
	Message string
	// Description says why it happened and what to do now.
	Description string
	// Data goes in the details of the status.
	Data msgpack.EncodedJSON
}

// Degraded returns an error that matches ErrDegraded and makes Runner.Report write w.
func (w Warning) Degraded() error { return errors.Mark(warningError{w}, ErrDegraded) }

// warningError carries a Warning to Runner.Report.
type warningError struct{ warning Warning }

// Error implements error.
func (e warningError) Error() string {
	return e.warning.Message + ": " + e.warning.Description
}

const (
	// SkipHold is how long a read task warns after it last skips a bad item.
	SkipHold = 5 * time.Second
	// skipRefresh is the shortest time between two changes of a skip warning, so that
	// a burst of bad items writes about one status each second.
	skipRefresh = time.Second
)

// Skips counts the items that a read task reads and the bad items that it skips. It
// gives a warning that holds the counts and the last reason, and that clears itself
// SkipHold after the last skip. The warning data holds the counts as stats.read and
// stats.skipped. Skips is not safe for concurrent use.
type Skips struct {
	noun          string
	read, skipped uint64
	// description is why the last item was skipped and where it was.
	description string
	until       time.Time
	// warning is the active warning, nil when none is active. It changes again at
	// refreshAt, when shown is no longer the skip count.
	warning   *Warning
	refreshAt time.Time
	shown     uint64
}

// NewSkips returns Skips that name the items with noun, such as "record". The warning
// adds an "s" to noun for any count other than one.
func NewSkips(noun string) *Skips { return &Skips{noun: noun} }

// Read counts an item read. Count each skipped item as read too.
func (s *Skips) Read() { s.read++ }

// Skip counts a skipped item. description says why it was skipped and where it was.
func (s *Skips) Skip(now time.Time, description string) {
	s.skipped++
	s.description = description
	s.until = now.Add(SkipHold)
}

// Warning returns the warning to show at now, or nil when the last skip was SkipHold
// or more ago. It returns the same warning until skipRefresh passes.
func (s *Skips) Warning(now time.Time) *Warning {
	if !now.Before(s.until) {
		s.warning = nil
		return nil
	}
	if s.warning != nil && (s.shown == s.skipped || now.Before(s.refreshAt)) {
		return s.warning
	}
	s.shown, s.refreshAt = s.skipped, now.Add(skipRefresh)
	noun := s.noun
	if s.read != 1 {
		noun += "s"
	}
	s.warning = &Warning{
		Message: fmt.Sprintf(
			"Skipped %s of %s %s", groupDigits(s.skipped), groupDigits(s.read), noun,
		),
		Description: s.description,
		Data: msgpack.EncodedJSON{
			"stats": map[string]any{"read": s.read, "skipped": s.skipped},
		},
	}
	return s.warning
}

// Deadline returns the next time at which Warning returns a new value without a new
// skip, and false when there is none.
func (s *Skips) Deadline() (time.Time, bool) {
	if s.warning == nil {
		return time.Time{}, false
	}
	if s.shown != s.skipped && s.refreshAt.Before(s.until) {
		return s.refreshAt, true
	}
	return s.until, true
}

// groupDigits formats n with a comma between groups of three digits.
func groupDigits(n uint64) string {
	digits := strconv.FormatUint(n, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := range len(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, digits[i])
	}
	return string(out)
}
