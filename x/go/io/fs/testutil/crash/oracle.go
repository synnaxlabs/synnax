// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash

import (
	"bytes"
	"fmt"
	"math"

	"github.com/synnaxlabs/x/set"
)

// attempt is one commit of the workload, and what the engine said about it.
type attempt struct {
	record Record
	// ackedAt is the count of events when the commit returned.
	ackedAt int
	// ack is the ack that the engine gave. It is zero while the commit runs, and for a
	// commit that failed.
	ack Ack
}

// rules sets what the oracle accepts from an engine that recovers.
type rules struct {
	// exemptFrom is the number of the event from which a Durable ack does not bind R1.
	exemptFrom int
	// refusalAllowed is true when the engine can refuse with its corruption error.
	refusalAllowed bool
	// unreadableAllowed is true when a record can be unreadable.
	unreadableAllowed bool
}

var strictRules = rules{exemptFrom: math.MaxInt}

// broken is a rule that an engine broke, with the reason.
type broken struct {
	rule   Rule
	detail string
}

// held applies R2 to the records that an engine returned for the attempts, and returns
// the set of commits that it holds.
func held(attempts []attempt, records []Record, r rules) (set.Set[uint64], *broken) {
	present, last := make(set.Set[uint64], len(records)), uint64(0)
	for _, rec := range records {
		if rec.ID == 0 || rec.ID > uint64(len(attempts)) {
			return nil, &broken{
				R2,
				fmt.Sprintf("commit %d is not in the workload", rec.ID),
			}
		}
		want := attempts[rec.ID-1].record.Payload
		switch {
		case rec.ID <= last:
			return nil, &broken{
				R2,
				fmt.Sprintf("commit %d is after commit %d", rec.ID, last),
			}
		case rec.Unreadable && !r.unreadableAllowed:
			return nil, &broken{R2, fmt.Sprintf("commit %d is unreadable", rec.ID)}
		case !rec.Unreadable && !bytes.Equal(rec.Payload, want):
			return nil, &broken{
				R2,
				fmt.Sprintf("commit %d has a wrong payload", rec.ID),
			}
		}
		present.Add(rec.ID)
		last = rec.ID
	}
	return present, nil
}

// checkLive applies R0 and R2 to the records of an engine that did not crash.
func checkLive(attempts []attempt, records []Record) *broken {
	present, b := held(attempts, records, strictRules)
	if b != nil {
		return b
	}
	for _, a := range attempts {
		if a.ack != 0 && !present.Contains(a.record.ID) {
			return &broken{
				R0,
				fmt.Sprintf("acked commit %d is not readable", a.record.ID),
			}
		}
	}
	return nil
}

// checkRecovered applies R1 to R3 to the records that an engine recovered from a crash
// image.
func checkRecovered(attempts []attempt, records []Record, r rules) *broken {
	present, b := held(attempts, records, r)
	if b != nil {
		return b
	}
	missing := uint64(0)
	for _, a := range attempts {
		if a.ack != 0 && !present.Contains(a.record.ID) {
			missing = a.record.ID
			break
		}
	}
	for _, rec := range records {
		if missing != 0 && rec.ID > missing {
			return &broken{R3, fmt.Sprintf(
				"acked commit %d is missing, and commit %d is not", missing, rec.ID,
			)}
		}
	}
	for _, a := range attempts {
		if a.ack == Durable && a.ackedAt < r.exemptFrom &&
			!present.Contains(a.record.ID) {
			return &broken{R1, fmt.Sprintf("durable commit %d is missing", a.record.ID)}
		}
	}
	return nil
}

// checkKept applies R1 to a second recovery, after a crash in the first: it holds each
// commit that the first held, and reads each one that the first read.
func checkKept(first, second []Record) *broken {
	again := make(map[uint64]Record, len(second))
	for _, rec := range second {
		again[rec.ID] = rec
	}
	for _, rec := range first {
		got, ok := again[rec.ID]
		switch {
		case !ok:
			return &broken{R1, fmt.Sprintf(
				"commit %d of the first recovery is missing after a crash in recovery",
				rec.ID,
			)}
		case got.Unreadable && !rec.Unreadable:
			return &broken{R1, fmt.Sprintf(
				"commit %d of the first recovery is unreadable after a crash in "+
					"recovery",
				rec.ID,
			)}
		}
	}
	return nil
}
