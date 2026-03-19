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

var _ = Describe("Service Passthrough", func() {
	Describe("Group", func() {
		It("Should return a valid group", func() {
			g := svc.Group()
			Expect(g.Key).ToNot(BeZero())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a channel created through the service", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			var retrieved svcChannel.Channel
			Expect(svc.NewRetrieve().WhereKeys(ch.Key()).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(ch.Name))
		})
	})

	Describe("NewObservable", func() {
		It("Should return a non-nil observable", func() {
			obs := svc.NewObservable()
			Expect(obs).ToNot(BeNil())
		})
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return a count", func() {
			count := svc.CountExternalNonVirtual()
			Expect(count).To(BeNumerically(">=", 0))
		})
	})

	Describe("Delete", func() {
		It("Should delete a channel by key", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(svc.Delete(ctx, ch.Key(), false)).To(Succeed())
		})
	})

	Describe("DeleteMany", func() {
		It("Should delete multiple channels by key", func() {
			channels := []svcChannel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			keys := []svcChannel.Key{channels[0].Key(), channels[1].Key()}
			Expect(svc.DeleteMany(ctx, keys, false)).To(Succeed())
		})
	})

	Describe("DeleteByName", func() {
		It("Should delete a channel by name", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(svc.DeleteByName(ctx, ch.Name, false)).To(Succeed())
		})
	})

	Describe("DeleteManyByNames", func() {
		It("Should delete multiple channels by name", func() {
			channels := []svcChannel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			names := []string{channels[0].Name, channels[1].Name}
			Expect(svc.DeleteManyByNames(ctx, names, false)).To(Succeed())
		})
	})

	Describe("Rename", func() {
		It("Should rename a channel", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			newName := channel.NewRandomName()
			Expect(svc.Rename(ctx, ch.Key(), newName, false)).To(Succeed())
			var retrieved svcChannel.Channel
			Expect(svc.NewRetrieve().WhereKeys(ch.Key()).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(newName))
		})
	})

	Describe("RenameMany", func() {
		It("Should rename multiple channels", func() {
			channels := []svcChannel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			newNames := []string{channel.NewRandomName(), channel.NewRandomName()}
			keys := []svcChannel.Key{channels[0].Key(), channels[1].Key()}
			Expect(svc.RenameMany(ctx, keys, newNames, false)).To(Succeed())
			var r0 svcChannel.Channel
			Expect(svc.NewRetrieve().WhereKeys(keys[0]).Entry(&r0).Exec(ctx, nil)).To(Succeed())
			Expect(r0.Name).To(Equal(newNames[0]))
		})
	})

	Describe("MapRename", func() {
		It("Should rename channels via old-to-new name map", func() {
			ch := svcChannel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			newName := channel.NewRandomName()
			Expect(svc.MapRename(ctx, map[string]string{ch.Name: newName}, false)).To(Succeed())
			var retrieved svcChannel.Channel
			Expect(svc.NewRetrieve().WhereKeys(ch.Key()).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(newName))
		})
	})
})
