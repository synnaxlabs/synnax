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
	"github.com/synnaxlabs/cesium/internal/testutil/opfs"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Switch failure check", func() {
	It("Should not switch to the next log after the frame before the switch fails its "+
		"sync", func() {
		fs := opfs.New(xfs.NewMem(), "write", "sync", "syncdir")
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		calls := &switches{}
		prepared := make(chan uint64, 4)
		var l *log.Log
		l = MustSucceed(log.Start(log.Config{
			FS:        fs,
			Executor:  goExec,
			OnDurable: noDurable,
			OnRefused: func([]log.Record, error) {},
			Prepare: func(next uint64) {
				calls.prepare(next)
				go func() {
					l.Prepared(next, log.Prepare(fs, next, logSize))
					prepared <- next
				}()
			},
			OnSwitched: calls.onSwitched,
			DB:         db,
			Generation: generation,
			Sequence:   5,
		}))
		ran := make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() { ran <- l.Run(ctx) }()
		Eventually(prepared).Should(Receive(Equal(uint64(generation + 1))))
		// Two frames end past half, so the frame of a small entry and a quarter crosses
		// three quarters.
		for range 2 {
			MustSucceed(l.Append(bigEntry(logSize / 4)).Wait())
		}
		fs.Hold()
		first := l.Append(of(&log.Drop{Key: 1}))
		Eventually(fs.Held()).Should(Receive(Equal(logName)))
		small := l.Append(of(&log.Drop{Key: 2}))
		quarter := l.Append(bigEntry(logSize / 4))
		eio := errors.New("input/output error")
		fs.FailSync(logName, eio)
		fs.Free()
		Expect(first.Wait()).To(Equal(uint64(7)))
		failed := SatisfyAll(
			MatchError(eio),
			MatchError(ContainSubstring(
				"log 3 failed, and no entry is durable until a restart",
			)),
		)
		Expect(small.Wait()).Error().To(failed)
		Expect(quarter.Wait()).Error().To(failed)
		Expect(l.Close()).To(failed)
		Eventually(ran).Should(Receive())
		_, switched := calls.get()
		Expect(switched).To(BeEmpty())
	})
})
