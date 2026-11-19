// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashcheck_test

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

const keys = 32

func runLog(ctx context.Context, o opts, p crash.Profile, key uint64) error {
	return crash.Run(ctx, crash.Config{
		RunKey:  key,
		Open:    openLog(o),
		Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
		Commits: 40,
		Stride:  1,
		Profile: p,
	})
}

// caught returns the violations of the log o over 32 run keys of profile p.
func caught(ctx context.Context, o opts, p crash.Profile) []string {
	var found []string
	for key := range uint64(keys) {
		if err := runLog(ctx, o, p, key); err != nil {
			found = append(found, err.Error())
		}
	}
	return found
}

var safeProfiles = []crash.Profile{
	crash.Strict,
	crash.Partial,
	crash.SyncErrors,
	crash.FullDisk,
	crash.OneTimeReadFaults,
	crash.Mixed,
	crash.NewCorruption,
}

var _ = Describe("Oracle", func() {
	It("Should pass the correct log", func(ctx SpecContext) {
		for _, p := range safeProfiles {
			Expect(caught(ctx, opts{}, p)).To(BeEmpty(), "profile %d", p)
		}
	})

	It("Should catch a log that acks Durable before its sync", func(ctx SpecContext) {
		for _, p := range safeProfiles {
			found := caught(ctx, opts{lazySync: true}, p)
			fmt.Fprintf(
				GinkgoWriter,
				"profile %d: %d of %d keys\n",
				p,
				len(found),
				keys,
			)
		}
		Expect(caught(ctx, opts{lazySync: true}, crash.Strict)).ToNot(BeEmpty())
	})

	It("Should pass a correct log that reads a record in two calls",
		func(ctx SpecContext) {
			o := opts{splitRead: true, tries: 3}
			Expect(caught(ctx, o, crash.OneTimeReadFaults)).To(BeEmpty())
		},
	)

	It("Should pass a correct log that stops reads after a sync error",
		func(ctx SpecContext) {
			o := opts{deadReads: true}
			Expect(caught(ctx, o, crash.SyncErrors)).To(BeEmpty())
			Expect(caught(ctx, o, crash.FullDisk)).To(BeEmpty())
		},
	)

	It("Should catch a log that gives no ack and no error", func(ctx SpecContext) {
		Expect(caught(ctx, opts{noAck: true}, crash.Strict)).ToNot(BeEmpty())
	})

	DescribeTable("Should catch a log that refuses an image with no flipped bit",
		func(ctx SpecContext, p crash.Profile) {
			err := crash.Run(ctx, crash.Config{
				RunKey:  0,
				Open:    openLog(opts{refuse: true}),
				Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
				Commits: 1,
				Stride:  1,
				Profile: p,
			})
			Expect(err).To(HaveOccurred())
		},
		Entry("OldCorruption", crash.OldCorruption),
	)

	It(
		"Should catch a log whose recovery cuts a durable record",
		func(ctx SpecContext) {
			disk := must(fault.New(fault.Config{}))
			open := openLog(opts{cutBack: true})
			e := must(open(ctx, disk))
			for id := range uint64(2) {
				r := crash.Record{ID: id + 1, Payload: []byte("durable")}
				Expect(e.Commit(ctx, r, crash.Durable)).To(Equal(crash.Durable))
			}
			Expect(e.Close()).To(Succeed())
			f := must(disk.Open("log", os.O_RDWR|os.O_APPEND))
			must(f.Write([]byte("torn")))
			Expect(f.Sync()).To(Succeed())
			Expect(f.Close()).To(Succeed())
			e = must(open(ctx, disk))
			Expect(must(e.Recovered(ctx))).To(HaveLen(2))
			Expect(e.Close()).To(Succeed())
			e = must(open(ctx, disk))
			Expect(
				must(e.Recovered(ctx)),
			).To(HaveLen(1), "the first recovery cut commit 2")
			Expect(e.Close()).To(Succeed())

			Expect(caught(ctx, opts{cutBack: true}, crash.Partial)).ToNot(BeEmpty())
		},
	)

	It("Should give equal results for equal run keys", func(ctx SpecContext) {
		for _, c := range []struct {
			o opts
			p crash.Profile
		}{
			{opts{lazySync: true}, crash.Mixed},
			{opts{splitRead: true, tries: 3}, crash.OneTimeReadFaults},
			{opts{}, crash.OldCorruption},
			{opts{}, crash.Partial},
		} {
			Expect(caught(ctx, c.o, c.p)).To(Equal(caught(ctx, c.o, c.p)))
		}
	})
})
