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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

var errScript = errors.New("script: no recovery")

// script runs steps on the first disk. Its recovery always fails, so a run fails at the
// first crash image and passes when the steps make fewer events than the stride.
type script struct{}

func (script) Commit(context.Context, crash.Record, crash.Ack) (crash.Ack, error) {
	return 0, errScript
}

func (script) Recovered(context.Context) ([]crash.Record, error) { return nil, nil }

func (script) Close() error { return nil }

func openScript(steps func(fs xfs.FS) error) func(
	context.Context,
	xfs.FS,
) (crash.Engine, error) {
	first := true
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		if !first {
			return nil, errScript
		}
		first = false
		return script{}, steps(fs)
	}
}

// events returns the count of events that steps make, as Run sees it.
func events(ctx context.Context, steps func(fs xfs.FS) error) int {
	count := 0
	for stride := 1; stride < 64; stride++ {
		err := crash.Run(ctx, crash.Config{
			Open:    openScript(steps),
			Commits: 1,
			Stride:  stride,
			Profile: crash.Strict,
		})
		if err == nil {
			return count
		}
		var v *crash.Violation
		Expect(errors.As(err, &v)).To(BeTrue(), "%v", err)
		Expect(v.Event).To(Equal(stride))
		count = stride
	}
	return count
}

var _ = Describe("Counter", func() {
	It("Should count each changing call once and no read", func(ctx SpecContext) {
		Expect(events(ctx, func(fs xfs.FS) error {
			sub := must(fs.Sub("d"))
			f := must(sub.Open("a", os.O_CREATE|os.O_RDWR))
			must(f.Write([]byte("abc")))
			must(f.ReadAt(make([]byte, 3), 0))
			Expect(f.Sync()).To(Succeed())
			g := must(sub.Open("a", os.O_CREATE|os.O_RDWR))
			again := must(fs.Sub("d"))
			must(f.WriteAt([]byte("x"), 3))
			Expect(again.SyncDir("")).To(Succeed())
			Expect(again.Rename("a", "b")).To(Succeed())
			Expect(g.Truncate(1)).To(Succeed())
			must(again.List(""))
			must(again.Exists("b"))
			must(again.Stat("b"))
			must(f.Stat())
			Expect(again.Remove("b")).To(Succeed())
			Expect(f.Close()).To(Succeed())
			return g.Close()
		})).To(Equal(9))
	})
})

func must[T any](v T, err error) T {
	GinkgoHelper()
	Expect(err).ToNot(HaveOccurred())
	return v
}
