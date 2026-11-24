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
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	. "github.com/synnaxlabs/x/testutil"
)

// Each entry holds on the pointer log, where a commit is durable once its writer
// closes: a refusal is allowed only where the profile allows one, and only with the
// corruption error.
var _ = DescribeTable(
	"Crash harness, pointer log",
	func(ctx SpecContext, ack crash.Ack, profile crash.Profile, ordered bool) {
		MustSucceed(crash.Run(ctx, crash.Config{
			Open: openCrashEngine(
				ack,
				cesium.WithPointerLog,
				func() cesium.Option { return cesium.WithRandom(rand.NewPCG(1, 2)) },
				func() cesium.Option { return pcgIdentity(1, 2) },
			),
			Commits:        8,
			Stride:         1,
			Profile:        profile,
			EntriesOrdered: ordered,
			Corrupt: func(err error) bool {
				return errors.Is(err, place.ErrCorrupt)
			},
		}))
	},
	Entry("Durable, Strict", crash.Durable, crash.Strict, false),
	Entry("Durable, SyncErrors", crash.Durable, crash.SyncErrors, false),
	Entry("Durable, FullDisk", crash.Durable, crash.FullDisk, false),
	Entry("Durable, Partial", crash.Durable, crash.Partial, false),
	Entry("Durable, OldCorruption", crash.Durable, crash.OldCorruption, false),
	Entry("Durable, NewCorruption", crash.Durable, crash.NewCorruption, false),
	Entry("Durable, OneTimeReadFaults", crash.Durable, crash.OneTimeReadFaults, false),
	Entry(
		"Durable, RepeatedReadFaults",
		crash.Durable,
		crash.RepeatedReadFaults,
		false,
	),
	Entry("Durable, Mixed", crash.Durable, crash.Mixed, false),
	Entry("Durable, Mixed, ordered", crash.Durable, crash.Mixed, true),
	Entry("Visible, Strict", crash.Visible, crash.Strict, false),
	Entry("Visible, FullDisk", crash.Visible, crash.FullDisk, false),
	Entry("Visible, Partial", crash.Visible, crash.Partial, false),
	Entry("Visible, Mixed", crash.Visible, crash.Mixed, false),
)
