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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: switch", func() {
	It("Should take a delete that fits in the last bytes of a log that cannot "+
		"switch, after another entry of its batch", func() {
		fs := newOpFS(xfs.NewMem())
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		var l *log.Log
		l = MustSucceed(log.Start(log.Config{
			FS:        fs,
			Executor:  goExec,
			OnDurable: noDurable,
			Prepare: func(next uint64) {
				l.Prepared(next, errors.New("no space left on device"))
			},
			OnRefused:  func([]log.Record, error) {},
			Switched:   func(uint64) {},
			DB:         db,
			Generation: generation,
			Sequence:   5,
		}))
		ran := make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { ran <- l.Run(ctx) }()
		DeferCleanup(func() {
			fs.Unhold()
			Expect(l.Close()).To(Succeed())
			Eventually(ran).Should(Receive(BeNil()))
			cancel()
		})
		const (
			limit = logSize - 2*log.BlockSize
			x     = 200_000
		)
		// A ends past three quarters, so that X ends 1,000 bytes before the limit of
		// an entry with no Commit.
		a := limit - 1_000 - (32 + x) - (log.FramesStart + 32)
		release := fs.Hold()
		first := l.Append(bigEntry(a))
		Eventually(fs.held).Should(Receive())
		second := l.Append(bigEntry(x))
		del := l.Append(of(&log.Drop{Key: 1}))
		fs.Unhold()
		release <- struct{}{}
		MustSucceed(first.Wait())
		MustSucceed(second.Wait())
		frames := walk(fileBytes(fs))
		Expect(len(frames)).To(BeNumerically(">=", 2))
		end := int64(log.FramesStart) + int64(frames[0].Size()+frames[1].Size())
		Expect(end+32+int64(len(entryOf(&log.Drop{Key: 1})))).
			To(BeNumerically("<=", limit), "the delete fits after X")
		Expect(del.Wait()).To(Equal(uint64(7)))
	})
})
