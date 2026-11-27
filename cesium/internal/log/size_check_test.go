// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Log size check", func() {
	start := func(size int64) error {
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, generation, size)).To(Succeed())
		l, err := log.Start(log.Config{
			FS:         fs,
			Executor:   goExec,
			OnDurable:  noDurable,
			OnRefused:  func([]log.Record, error) {},
			Prepare:    func(uint64) {},
			Switched:   func(uint64) {},
			DB:         db,
			Generation: generation,
			Sequence:   1,
		})
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ran := make(chan error, 1)
		go func() { ran <- l.Run(ctx) }()
		return errors.Join(l.Close(), <-ran)
	}

	DescribeTable("Should refuse to start a log under 8 MiB", func(size int64) {
		Expect(start(size)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(fmt.Sprintf(
				"log file of %d bytes is not a whole number of blocks of at least "+
					"8388608",
				size,
			))),
		))
	},
		Entry("of 3 MiB", int64(3<<20)),
		Entry("of 4 MiB, the smallest before", int64(4<<20)),
		Entry("of one block under 8 MiB", int64(8<<20-log.BlockSize)),
	)

	It("Should start a log of 8 MiB", func() {
		Expect(start(8 * log.DeleteReserve)).To(Succeed())
	})
})
