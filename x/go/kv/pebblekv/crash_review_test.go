// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"context"

	"github.com/cockroachdb/pebble/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// counted counts the Durable acks of the engine that runs the workload.
type counted struct {
	crash.Engine
	acks *int
}

func (c counted) Commit(
	ctx context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	ack, err := c.Engine.Commit(ctx, r, want)
	if err == nil && ack == crash.Durable {
		*c.acks++
	}
	return ack, err
}

var _ = Describe("Crash adapter review", func() {
	It("Should not take a panic that is not a Fatalf for a stop", func() {
		f := func() (err error) {
			defer stopped(&err)
			var m map[string]int
			m["a"] = 1
			return nil
		}
		Expect(f).To(Panic())
	})

	DescribeTable("Should ack a durable commit in each holding entry",
		func(ctx SpecContext, profile crash.Profile, runKey uint64) {
			acks := 0
			open := func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				e, err := openCrashEngine(ctx, fs)
				if err != nil {
					return nil, err
				}
				return counted{Engine: e, acks: &acks}, nil
			}
			_, err := crash.Run(ctx, crash.Config{
				RunKey: runKey,
				Open:   open,
				Corrupt: func(err error) bool {
					return errors.Is(err, pebble.ErrCorruption)
				},
				Commits: 20,
				Stride:  1,
				Profile: profile,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(acks).To(BeNumerically(">", 0))
		},
		Entry("Strict", crash.Strict, uint64(0)),
		Entry("SyncErrors", crash.SyncErrors, uint64(0)),
		Entry("FullDisk", crash.FullDisk, uint64(0)),
		Entry("NewCorruption", crash.NewCorruption, uint64(0)),
		Entry("Partial, run key 1", crash.Partial, uint64(1)),
		Entry("Mixed, run key 1", crash.Mixed, uint64(1)),
	)
})
