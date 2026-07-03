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
	"fmt"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Service", func() {
	Describe("Group", func() {
		It("Should return a valid group", func(ctx SpecContext) {
			g := svc.Group()
			Expect(g.Key).ToNot(BeZero())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a channel created through the service", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(ch.Name))
		})
	})

	Describe("Observe", func() {
		It("Should return a non-nil observable", func(ctx SpecContext) {
			obs := svc.Observe()
			Expect(obs).ToNot(BeNil())
		})
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := svc.Observe().OnChange(func(ctx context.Context, _ gorp.TxReader[channel.Key, channel.Channel]) {
				called.Store(true)
			})
			DeferCleanup(disconnect)
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})

	Describe("Create", func() {
		It("Should infer Int64 DataType from integer arithmetic", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should infer Float64 DataType from a channel reference expression", func(ctx SpecContext) {
			base := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &base)).To(Succeed())
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: fmt.Sprintf("return %s * 2.0", base.Name),
				Virtual:    true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should infer Float64 DataType from float literal expression", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1.5 + 2.5",
				Virtual:    true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should overwrite caller-provided DataType with inferred type", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.StringT,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should return a parse error for an invalid expression", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return invalid_syntax {{",
				Virtual:    true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(MatchError(
				ContainSubstring("extraneous input '{'"),
			))
		})

		It("Should not modify DataType for non-calculated channels", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.TimeStampT))
		})
	})

	Describe("CreateMany", func() {
		It("Should infer types for calculated channels and pass through non-calculated", func(ctx SpecContext) {
			nonCalc := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &nonCalc)).To(Succeed())
			Expect(nonCalc.DataType).To(Equal(telem.Float64T))

			channels := []channel.Channel{
				{
					Name:       channel.NewRandomName(),
					Expression: "return 1 + 1",
					Virtual:    true,
				},
				{
					Name:       channel.NewRandomName(),
					Expression: "return 1.5 + 2.5",
					Virtual:    true,
				},
			}
			Expect(svc.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Float64T))
		})

		It("Should handle an empty slice without error", func(ctx SpecContext) {
			channels := []channel.Channel{}
			Expect(svc.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
		})

		It("Should resolve cross-references within the same batch", func(ctx SpecContext) {
			firstName := channel.NewRandomName()
			channels := []channel.Channel{
				{
					Name:       firstName,
					Expression: "return 1 + 1",
					Virtual:    true,
				},
				{
					Name:       channel.NewRandomName(),
					Expression: fmt.Sprintf("return %s * 2", firstName),
					Virtual:    true,
				},
			}
			Expect(svc.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Int64T))
		})
	})

	Describe("NewWriter", func() {
		It("Should create a writer that infers types for calculated channels", func(ctx SpecContext) {
			w := svc.NewWriter(nil)
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(w.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})
	})
})
