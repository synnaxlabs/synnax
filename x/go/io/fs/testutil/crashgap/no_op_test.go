// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashgap_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// refuser fails every commit, though no fault makes it fail.
type refuser struct{}

func (refuser) Commit(context.Context, crash.Record, crash.Ack) (crash.Ack, error) {
	return 0, errors.New("refused")
}

func (refuser) Recovered(context.Context) ([]crash.Record, error) { return nil, nil }

func (refuser) Close() error { return nil }

var _ = Describe("Engines that do nothing", func() {
	It(
		"Should fail an engine that fails each commit on a disk with no faults",
		func(ctx SpecContext) {
			_, err := crash.Run(ctx, crash.Config{
				RunKey:  1,
				Open:    func(context.Context, xfs.FS) (crash.Engine, error) { return refuser{}, nil },
				Commits: 20,
				Stride:  1,
				Profile: crash.Strict,
			})
			Expect(err).To(HaveOccurred())
		},
	)

	DescribeTable("Should fail an engine that never opens, when no fault fired",
		func(ctx SpecContext, profile crash.Profile) {
			_, err := crash.Run(ctx, crash.Config{
				RunKey: 1,
				Open: func(context.Context, xfs.FS) (crash.Engine, error) {
					return nil, errors.New("broken")
				},
				Commits: 20,
				Stride:  1,
				Profile: profile,
			})
			Expect(err).To(HaveOccurred())
		},
		Entry("full disk", crash.FullDisk),
		Entry("sync errors", crash.SyncErrors),
	)
})
