// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// slowSetKV holds the first Set after holding turns true until release closes, as a
// slow write of one of two concurrent Allocates.
type slowSetKV struct {
	kv.ReadWriter
	holding *atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (s slowSetKV) Set(ctx context.Context, key, value []byte) error {
	if s.holding.CompareAndSwap(true, false) {
		close(s.entered)
		<-s.release
	}
	return s.ReadWriter.Set(ctx, key, value)
}

var _ = Describe("Issued with two concurrent Allocates", func() {
	It("Should not report a key that storage does not hold", func(ctx SpecContext) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		slow := slowSetKV{
			holding: &atomic.Bool{},
			entered: make(chan struct{}),
			release: make(chan struct{}),
		}
		slow.ReadWriter = store.KV
		host := node.Key(1)
		open := func() *channel.Service {
			GinkgoHelper()
			return MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
				HostKey: host,
				KV:      slow,
				TS:      store.TS,
			}))
		}
		ch := channel.Channel{
			Name:        "pressure",
			DataType:    telem.Float32T,
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
		svc := open()
		MustSucceed(svc.Allocate(ctx, []channel.Channel{ch}))
		slow.holding.Store(true)
		allocate := func() <-chan []channel.Channel {
			done := make(chan []channel.Channel, 1)
			go func() {
				defer GinkgoRecover()
				done <- MustSucceed(svc.Allocate(ctx, []channel.Channel{ch}))
			}()
			return done
		}
		firstDone := allocate()
		<-slow.entered
		secondDone := allocate()
		// A second Allocate that does not wait for the first writes its counter first.
		var second []channel.Channel
		select {
		case second = <-secondDone:
		case <-time.After(200 * time.Millisecond):
		}
		close(slow.release)
		first := (<-firstDone)[0]
		if second == nil {
			second = <-secondDone
		}
		last := second[0]
		Expect(last.LocalKey).To(Equal(first.LocalKey + 1))
		issued := svc.Issued(last.Key())
		reopened := open()
		again := MustSucceed(reopened.Allocate(ctx, []channel.Channel{ch}))[0]
		Expect(issued).To(BeTrue())
		Expect(again.LocalKey).To(BeNumerically(">", last.LocalKey))
	})
})
