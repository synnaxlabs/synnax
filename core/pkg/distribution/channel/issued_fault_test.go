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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var errDiskFull = errors.New("disk full")

// failingSetKV fails every Set while failing is true.
type failingSetKV struct {
	kv.ReadWriter
	failing *atomic.Bool
}

func (f failingSetKV) Set(ctx context.Context, key, value []byte) error {
	if f.failing.Load() {
		return errDiskFull
	}
	return f.ReadWriter.Set(ctx, key, value)
}

var _ = Describe("Issued after a failed counter write", func() {
	It("Should not report a key that a failed Allocate never issued", func(
		ctx SpecContext,
	) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		failing := &atomic.Bool{}
		host := node.Key(1)
		open := func() *channel.Service {
			GinkgoHelper()
			return MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
				HostKey: host,
				KV:      failingSetKV{ReadWriter: store.KV, failing: failing},
			}))
		}
		ch := channel.Channel{
			Name:        "pressure",
			DataType:    telem.Float32T,
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
		svc := open()
		first := MustSucceed(svc.Allocate(ctx, []channel.Channel{ch}))[0]
		failing.Store(true)
		Expect(svc.Allocate(ctx, []channel.Channel{ch})).Error().
			To(MatchError(errDiskFull))
		failing.Store(false)
		next := channel.NewKey(node.KeyFree, first.LocalKey+1)
		issued := svc.Issued(next)
		// After a restart, the counter gives next to a new channel, so a weak link
		// that a write accepted while Issued(next) was true binds to it.
		reopened := open()
		again := MustSucceed(reopened.Allocate(ctx, []channel.Channel{ch}))[0]
		Expect(again.Key()).To(Equal(next))
		Expect(issued).To(BeFalse())
	})
})
