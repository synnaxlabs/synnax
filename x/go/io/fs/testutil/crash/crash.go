// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package crash runs a storage engine on a fault.FS, crashes it after each call that
// changes the disk, and checks what the engine recovers against the acks that it gave.
package crash

import (
	"context"
	"fmt"

	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

// Ack is the promise that an engine gives for a commit.
type Ack uint8

const (
	// Visible means that a read sees the commit now, and that a crash can lose it.
	Visible Ack = iota + 1
	// Durable means that the commit survives a crash.
	Durable
)

// Record is one commit of the workload.
type Record struct {
	// ID is the place of the commit in the workload, from 1.
	ID uint64
	// Payload holds the bytes of the commit.
	Payload []byte
	// Unreadable is true when the engine holds the commit and a read of it fails with a
	// corruption error.
	Unreadable bool
}

// Engine is the store under test. An adapter maps each record to the data of the store.
type Engine interface {
	// Commit stores r and asks for the ack want. It returns the ack that the engine
	// gives, which can be the other level.
	Commit(ctx context.Context, r Record, want Ack) (Ack, error)
	// Recovered returns the commits that the engine holds, in commit order.
	Recovered(ctx context.Context) ([]Record, error)
	// Close stops the engine.
	Close() error
}

// Profile names the faults of a run, and with them the rules that the oracle applies.
type Profile uint8

const (
	// Strict crashes keep nothing that is not durable.
	Strict Profile = iota + 1
	// Partial crashes keep a part of the unsynced data and directory changes, with torn
	// blocks.
	Partial
	// SyncErrors fails a part of the Sync and SyncDir calls of the workload.
	SyncErrors
	// FullDisk gives the workload a disk that fills.
	FullDisk
	// OneTimeReadFaults gives the engine that recovers read faults that a second read
	// does not repeat. The engine must not refuse.
	OneTimeReadFaults
	// OldCorruption flips, for good, a bit of a durable byte in any file. The byte is
	// not in the unit: the bytes that the newest Sync with a changed block made durable
	// from writes. The engine can refuse with its corruption error, or return
	// unreadable records. It must not lose a Durable ack.
	OldCorruption
	// NewCorruption makes one 4 KiB block of the unit zeros for good, as a block that
	// the disk never wrote is. No later Sync confirms the unit. The engine can lose
	// each commit whose Durable ack came after the first Sync that changed the first
	// block of the written range that holds the stale block. It can also refuse with
	// its corruption error.
	NewCorruption
	// Mixed joins the faults of Partial, SyncErrors, FullDisk, and OneTimeReadFaults.
	Mixed
	// RepeatedReadFaults gives the engine that recovers read faults that each read can
	// get, retries too. A cluster is the faults whose reads overlap in one file. After
	// a cluster of two or more faults, the engine can refuse with its corruption error,
	// return unreadable records, or lose the commits from the first Sync that changed
	// the earliest block that a fault of the cluster hit. It must never return wrong
	// data or a hole.
	RepeatedReadFaults
)

// Rule names one rule of the oracle.
type Rule string

const (
	// R0 says that the engine works before a crash. On a disk that fails no call, it
	// opens, takes each commit with a Visible or a Durable ack, and closes. It shows
	// each commit that it acked.
	R0 Rule = "R0"
	// R1 says that each commit with a Durable ack is in the recovered list. After a
	// crash in the recovery, the list holds each commit that the first recovery held
	// again, and each one that the first could read is readable.
	R1 Rule = "R1"
	// R2 says that the recovered list holds only commits of the workload, each once, in
	// commit order, each with its whole payload.
	R2 Rule = "R2"
	// R3 says that the recovered list has no hole: when an acked commit is missing, no
	// later commit is in the list, acked or not.
	R3 Rule = "R3"
	// R4 says that the engine opens, reads, and closes on each crash image, and again
	// after a crash in that recovery. It can refuse, with the corruption error, only an
	// image that OldCorruption or NewCorruption changed, on the first open, or after
	// read faults that RepeatedReadFaults allows.
	R4 Rule = "R4"
	// R5 says that the state that each recovery returns keeps the invariants that the
	// Audit of the engine checks.
	R5 Rule = "R5"
)

// Auditor is an Engine that checks its own invariants. Run calls Audit after each
// recovery that returned its records, and an error breaks R5. Audit checks only what
// Recovered read, and reads no disk, so a read fault reaches only Recovered.
type Auditor interface {
	Engine
	Audit(ctx context.Context) error
}

// Violation is the error of a run that broke a rule. The run key and the event replay
// it.
type Violation struct {
	// Detail says what the oracle found.
	Detail string
	// Rule is the rule that the engine broke.
	Rule Rule
	// RunKey is the run key of the run.
	RunKey uint64
	// Event is the count of disk changes before the crash. Zero is the engine before
	// any crash.
	Event int
	// Profile is the profile of the run.
	Profile Profile
	// ReadFaults is the count of read faults in the recovery that broke the rule, and
	// Clustered is the count in the largest cluster of them: faults whose reads overlap
	// in one file.
	ReadFaults, Clustered int
}

// Error implements error.
func (v *Violation) Error() string {
	return fmt.Sprintf(
		"crash: %s broken at event %d of run key %d, profile %d, with %d read faults, "+
			"%d in one cluster: %s",
		v.Rule, v.Event, v.RunKey, v.Profile, v.ReadFaults, v.Clustered, v.Detail,
	)
}

// Result counts what a run checked.
type Result struct {
	// Recoveries counts the opens of the engine on crash images.
	Recoveries int
	// Refusals counts the recoveries that refused, as the rules allow.
	Refusals int
	// Losses counts the recoveries that lost an acked commit, as the rules allow.
	Losses int
	// ReadFaults counts the read faults that the disks of the recoveries gave.
	ReadFaults int
}

// Config sets one run of the harness.
type Config struct {
	// Open opens the engine on fs. Run calls it for the workload, and again for each
	// crash image.
	Open func(ctx context.Context, fs xfs.FS) (Engine, error)
	// Corrupt reports if err is the corruption error of the engine. Nil means that no
	// error is.
	Corrupt func(err error) bool
	// RunKey fixes the workload, the ack levels, the faults, and the crash images.
	RunKey uint64
	// Commits is the count of commits in the workload.
	Commits int
	// Stride is the count of events between two crash images. One checks each event.
	Stride int
	// Profile sets the faults and the rules.
	Profile Profile
	// EntriesOrdered makes the directory changes that a crash keeps a prefix of the
	// changes in the order they were made, as a metadata journal keeps them. Only
	// Partial and Mixed lose directory changes, so it changes only those profiles.
	EntriesOrdered bool
}

func (c Config) validate() error {
	v := validate.New("crash.Config")
	v.NotNil("open", c.Open)
	v.Positive("commits", c.Commits)
	v.Positive("stride", c.Stride)
	v.Ternary(
		"profile",
		c.Profile < Strict || c.Profile > RepeatedReadFaults,
		"must be a profile",
	)
	return v.Error()
}
