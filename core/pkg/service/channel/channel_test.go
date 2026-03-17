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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	svcChannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should infer Int64 DataType from integer arithmetic", func() {
			ch := svcChannel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should infer Float64 DataType from a channel reference expression", func() {
			base := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			ch := svcChannel.Channel{
				Name:       channel.NewRandomName(),
				Expression: fmt.Sprintf("return %s * 2.0", base.Name),
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should infer Float64 DataType from float literal expression", func() {
			ch := svcChannel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1.5 + 2.5",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should overwrite caller-provided DataType with inferred type", func() {
			ch := svcChannel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.StringT,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should return a parse error for an invalid expression", func() {
			ch := svcChannel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return invalid_syntax {{",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(MatchError(
				ContainSubstring("extraneous input '{'"),
			))
		})

		It("Should not modify DataType for non-calculated channels", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.TimeStampT))
		})
	})

	Describe("CreateMany", func() {
		It("Should infer types for calculated channels and pass through non-calculated", func() {
			nonCalc := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &nonCalc)).To(Succeed())
			Expect(nonCalc.DataType).To(Equal(telem.Float64T))

			channels := []svcChannel.Channel{
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
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Float64T))
		})

		It("Should handle an empty slice without error", func() {
			channels := []svcChannel.Channel{}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
		})

		It("Should resolve cross-references within the same batch", func() {
			firstName := channel.NewRandomName()
			channels := []svcChannel.Channel{
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
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Int64T))
		})
	})
})

var _ = Describe("NewWriter", func() {
	It("Should create a writer that infers types for calculated channels", func() {
		w := svc.NewWriter(nil)
		ch := svcChannel.Channel{
			Name:       channel.NewRandomName(),
			Expression: "return 1 + 1",
			Virtual:    true,
		}
		Expect(w.Create(ctx, &ch)).To(Succeed())
		Expect(ch.DataType).To(Equal(telem.Int64T))
	})
})
