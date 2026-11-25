// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	. "github.com/synnaxlabs/x/testutil"
)

// runLog runs the harness on the pointer log, where a commit is durable once its writer
// closes, and returns its counts. A refusal is allowed only where the profile allows
// one, and only with the corruption error.
func runLog(
	ctx context.Context,
	ack crash.Ack,
	profile crash.Profile,
	ordered bool,
) crash.Result {
	GinkgoHelper()
	opts := []func() cesium.Option{
		cesium.WithPointerLog,
		func() cesium.Option { return cesium.WithRandom(rand.NewPCG(1, 2)) },
	}
	if faultsReads(profile) {
		opts = append(opts, cesium.WithReadsChecked)
	}
	return MustSucceed(crash.Run(ctx, crash.Config{
		Open:           openCrashEngine(ack, opts...),
		Commits:        8,
		Stride:         1,
		Profile:        profile,
		EntriesOrdered: ordered,
		Corrupt: func(err error) bool {
			return errors.Is(err, place.ErrCorrupt)
		},
	}))
}

// faultsReads reports whether profile corrupts or faults the bytes that a read returns.
// An unchecked read gives those bytes back as data, so the rule against wrong data
// holds under such a profile only with checked reads.
func faultsReads(profile crash.Profile) bool {
	switch profile {
	case crash.OldCorruption,
		crash.NewCorruption,
		crash.OneTimeReadFaults,
		crash.RepeatedReadFaults,
		crash.Mixed:
		return true
	case crash.Strict, crash.Partial, crash.SyncErrors, crash.FullDisk:
		return false
	}
	panic(fmt.Sprintf("crash profile %d", profile))
}

var _ = DescribeTable("Crash harness, pointer log",
	func(ctx SpecContext, ack crash.Ack, profile crash.Profile, ordered bool) {
		runLog(ctx, ack, profile, ordered)
	},
	Entry("Durable, Strict", crash.Durable, crash.Strict, false),
	Entry("Durable, Partial", crash.Durable, crash.Partial, false),
	Entry("Durable, OldCorruption", crash.Durable, crash.OldCorruption, false),
	Entry("Durable, NewCorruption", crash.Durable, crash.NewCorruption, false),
	Entry(
		"Durable, RepeatedReadFaults",
		crash.Durable,
		crash.RepeatedReadFaults,
		false,
	),
)

// Mixed joins the faults of Partial, SyncErrors, FullDisk, and OneTimeReadFaults, and
// stands in for their entries, so each of its runs must fire each of the four.
var _ = DescribeTable("Crash harness, pointer log, Mixed",
	func(ctx SpecContext, ack crash.Ack, ordered bool) {
		res := runLog(ctx, ack, crash.Mixed, ordered)
		Expect(res.Unsynced).To(BeNumerically(">", 0), "unsynced images")
		Expect(res.SyncErrors).To(BeNumerically(">", 0), "sync errors")
		Expect(res.FullDisk).To(BeNumerically(">", 0), "full disk")
		Expect(res.ReadFaults).To(BeNumerically(">", 0), "read faults")
	},
	Entry("Durable", crash.Durable, false),
	Entry("Durable, ordered", crash.Durable, true),
	Entry("Visible", crash.Visible, false),
)
