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
	"encoding/binary"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// blockingKV holds every Set until release closes, and signals entered on the first.
type blockingKV struct {
	kv.ReadWriter
	entered chan struct{}
	release chan struct{}
}

func (b *blockingKV) Set(ctx context.Context, key, value []byte) error {
	select {
	case b.entered <- struct{}{}:
	default:
	}
	<-b.release
	return b.ReadWriter.Set(ctx, key, value)
}

// validServiceConfig opens an in-memory node and returns a ServiceConfig backed by its
// host key, key-value store, and time-series database.
func validServiceConfig(ctx context.Context) channel.ServiceConfig {
	node := testutil.NewNode(ctx)
	return channel.ServiceConfig{
		HostKey: node.Identity.HostKey,
		KV:      node.DB,
		TS:      node.Storage.TS,
	}
}

var _ = Describe("ServiceConfig", Ordered, func() {
	var valid channel.ServiceConfig
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		valid = validServiceConfig(ctx)
	})

	Describe("Validate", func() {
		It("Should pass when all required fields are set", func() {
			Expect(valid.Validate()).To(Succeed())
		})
		DescribeTable("Should report each missing required field",
			func(field string) {
				Expect(channel.ServiceConfig{}.Validate()).To(
					MatchError(ContainSubstring(field)),
				)
			},
			Entry("host key", "host_key: must be non-zero"),
			Entry("kv", "kv: must be non-nil"),
			Entry("time-series database", "ts: must be non-nil"),
		)
	})

	Describe("Override", func() {
		It("Should carry the override's fields onto a zero config", func() {
			Expect(channel.ServiceConfig{}.Override(valid).Validate()).To(Succeed())
		})
	})
})

var _ = Describe("Service", Ordered, func() {
	var valid channel.ServiceConfig
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		valid = validServiceConfig(ctx)
	})

	It("Should open with a valid configuration", func(ctx SpecContext) {
		Expect(channel.NewService(ctx, valid)).ToNot(BeNil())
	})
	It("Should return an error with an invalid configuration", func(ctx SpecContext) {
		Expect(channel.NewService(ctx, channel.ServiceConfig{})).Error().To(
			MatchError(SatisfyAll(
				ContainSubstring("host_key: must be non-zero"),
				ContainSubstring("kv: must be non-nil"),
				ContainSubstring("ts: must be non-nil"),
			)),
		)
	})
})

var _ = Describe("Service key cap", func() {
	It("Should fail the allocation that passes the cap while another is flushing",
		func(ctx SpecContext) {
			cfg := validServiceConfig(ctx)
			key := []byte(cfg.HostKey.String() + ".distribution.channel.leasedCounter")
			var buf [8]byte
			binary.LittleEndian.PutUint64(buf[:], uint64(math.MaxUint20-1))
			Expect(cfg.KV.Set(ctx, key, buf[:])).To(Succeed())
			blocking := &blockingKV{
				ReadWriter: cfg.KV,
				entered:    make(chan struct{}, 1),
				release:    make(chan struct{}),
			}
			cfg.KV = blocking
			svc := MustSucceed(channel.NewService(ctx, cfg))
			allocate := func() error {
				_, err := svc.Allocate(ctx, []channel.Channel{
					{Name: "capped", DataType: telem.TimestampT, IsIndex: true},
				})
				return err
			}
			first, second := make(chan error, 1), make(chan error, 1)
			go func() { first <- allocate() }()
			Eventually(blocking.entered).Should(Receive())
			go func() { second <- allocate() }()
			Consistently(second, 100*time.Millisecond).ShouldNot(Receive())
			close(blocking.release)
			Eventually(first).Should(Receive(BeNil()))
			Eventually(second).Should(
				Receive(MatchError(ContainSubstring("maximum number of channels"))),
			)
		})
})
