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
	// OldCorruption flips bits in bytes that were durable before the last Sync began.
	// The engine can refuse with its corruption error, or return unreadable records.
	OldCorruption
	// NewCorruption flips bits in the bytes that the last Sync made durable. The
	// commits whose Durable ack came after that Sync began can be lost.
	NewCorruption
	// Mixed joins the faults of Partial, SyncErrors, FullDisk, and OneTimeReadFaults.
	Mixed
)

// Rule names one rule of the oracle.
type Rule string

const (
	// R0 says that the engine, before a crash, holds each commit that it acked.
	R0 Rule = "R0"
	// R1 says that each commit with a Durable ack is in the recovered list.
	R1 Rule = "R1"
	// R2 says that the recovered list holds only commits of the workload, each once, in
	// commit order, each with its whole payload.
	R2 Rule = "R2"
	// R3 says that the recovered list has no hole: when an acked commit is missing, no
	// later commit is in the list.
	R3 Rule = "R3"
	// R4 says that the engine opens and reads on each crash image. It can refuse only
	// under OldCorruption, and only with its corruption error.
	R4 Rule = "R4"
)

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
}

// Error implements error.
func (v *Violation) Error() string {
	return fmt.Sprintf(
		"crash: %s broken at event %d of run key %d, profile %d: %s",
		v.Rule, v.Event, v.RunKey, v.Profile, v.Detail,
	)
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
}

func (c Config) validate() error {
	v := validate.New("crash.Config")
	v.NotNil("open", c.Open)
	v.Positive("commits", c.Commits)
	v.Positive("stride", c.Stride)
	v.Ternary("profile", c.Profile < Strict || c.Profile > Mixed, "must be a profile")
	return v.Error()
}
