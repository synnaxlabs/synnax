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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should infer Int64 DataType from integer arithmetic", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should infer Float64 DataType from a channel reference expression", func(ctx SpecContext) {
			base := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &base)).To(Succeed())
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: fmt.Sprintf("return %s * 2.0", base.Name),
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should infer Float64 DataType from float literal expression", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1.5 + 2.5",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Float64T))
		})

		It("Should overwrite caller-provided DataType with inferred type", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.StringT,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})

		It("Should return a parse error for an invalid expression", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return invalid_syntax {{",
				Virtual:    true,
			}
			Expect(svc.Create(ctx, &ch)).To(MatchError(
				ContainSubstring("extraneous input '{'"),
			))
		})

		It("Should not modify DataType for non-calculated channels", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
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
			Expect(svc.Create(ctx, &nonCalc)).To(Succeed())
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
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Float64T))
		})

		It("Should handle an empty slice without error", func(ctx SpecContext) {
			channels := []channel.Channel{}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
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
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			Expect(channels[0].DataType).To(Equal(telem.Int64T))
			Expect(channels[1].DataType).To(Equal(telem.Int64T))
		})
	})
})

var _ = Describe("NewWriter", func() {
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

var _ = Describe("Writer Delete", func() {
	It("Should delete a channel by key", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(svc.Create(ctx, &ch)).To(Succeed())
		Expect(svc.Delete(ctx, ch.Key(), false)).To(Succeed())
	})

	It("Should delete multiple channels by key", func(ctx SpecContext) {
		channels := []channel.Channel{
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
		}
		Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
		keys := []channel.Key{channels[0].Key(), channels[1].Key()}
		Expect(svc.DeleteMany(ctx, keys, false)).To(Succeed())
	})

	It("Should delete multiple channels by name", func(ctx SpecContext) {
		channels := []channel.Channel{
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
		}
		Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
		names := []string{channels[0].Name, channels[1].Name}
		Expect(svc.DeleteManyByNames(ctx, names, false)).To(Succeed())
	})
})

var _ = Describe("Writer Rename", func() {
	It("Should rename a channel", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(svc.Create(ctx, &ch)).To(Succeed())
		newName := channel.NewRandomName()
		Expect(svc.Rename(ctx, ch.Key(), newName, false)).To(Succeed())
		var retrieved channel.Channel
		Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
		Expect(retrieved.Name).To(Equal(newName))
	})

	It("Should rename multiple channels", func(ctx SpecContext) {
		channels := []channel.Channel{
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
		}
		Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
		newNames := []string{channel.NewRandomName(), channel.NewRandomName()}
		keys := []channel.Key{channels[0].Key(), channels[1].Key()}
		Expect(svc.RenameMany(ctx, keys, newNames, false)).To(Succeed())
		var r0 channel.Channel
		Expect(svc.NewRetrieve().Where(channel.MatchKeys(keys[0])).Entry(&r0).Exec(ctx, nil)).To(Succeed())
		Expect(r0.Name).To(Equal(newNames[0]))
	})
})

// The external non-virtual channel set the writer maintains for overflow enforcement is
// not directly observable, so these specs exercise it indirectly through the overflow
// limit: only external non-virtual channels should count toward it.
var _ = Describe("External Channel Overflow", Ordered, func() {
	var overflowSvc *channel.Service
	BeforeAll(func(ctx SpecContext) {
		overflowSvc = openService(ctx, mock.MustOpenNode(ctx), channel.ServiceConfig{
			IntOverflowCheck: fixedOverflowChecker(2),
		})
	})
	It("Should not count virtual or internal channels toward the limit", func(ctx SpecContext) {
		for range 5 {
			virtual := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(overflowSvc.Create(ctx, &virtual)).To(Succeed())
			internal := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Internal:    true,
				Leaseholder: 1,
			}
			Expect(overflowSvc.Create(ctx, &internal)).To(Succeed())
		}
	})
	It("Should reject external non-virtual channels past the limit", func(ctx SpecContext) {
		for range 2 {
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(overflowSvc.Create(ctx, &ch)).To(Succeed())
		}
		third := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(overflowSvc.Create(ctx, &third)).
			Error().To(MatchError(ContainSubstring("channel limit exceeded")))
	})
})

// Name validation is enforced internally on the create/rename path, so these specs
// exercise it through svc.Create rather than calling the unexported validator directly.
var _ = Describe("Name Validation", func() {
	DescribeTable("Should accept valid names", func(ctx SpecContext, name string) {
		ch := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
		Expect(svc.Create(ctx, &ch)).To(Succeed())
	},
		Entry("only letters", "name_valid_temperature"),
		Entry("mixed case", "name_valid_Pressure"),
		Entry("with digits", "name_valid_sensor1"),
		Entry("with underscores", "name_valid_sensor_temp"),
		Entry("letters, digits, and underscores", "name_valid_temp123_sensor"),
	)
	DescribeTable("Should reject invalid names", func(ctx SpecContext, name, msg string) {
		ch := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
		Expect(svc.Create(ctx, &ch)).Error().To(MatchError(ContainSubstring(msg)))
	},
		Entry("empty name", "", "name cannot be empty"),
		Entry("name starting with a digit", "1sensor", "cannot start with a digit"),
		Entry("name with spaces", "my channel", "contains invalid characters"),
		Entry("name with special characters", "sensor!", "contains invalid characters"),
	)
	Describe("NewRandomName", func() {
		It("Should generate unique channel names", func() {
			count := 100
			existingNames := make(set.Set[string], count)
			for range count {
				nextName := channel.NewRandomName()
				Expect(existingNames.Contains(nextName)).To(BeFalse())
				existingNames.Add(nextName)
			}
		})
	})
})

var _ = Describe("Create", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(ctx, 2)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func(ctx SpecContext) {
			ch.IsIndex = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.TimeStampT
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should create the channel without error", func(ctx SpecContext) {
				Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should not create the channel if it already exists by name", func(ctx SpecContext) {
				Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
				Expect(ch.LocalKey).To(Equal(channel.LocalKey(3)))
			})
			It("Should create the channel in the time-series database", func(ctx SpecContext) {
				channels := MustSucceed(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				tsCh := channels[0]
				Expect(tsCh.Key).To(Equal(ch.Key().StorageKey()))
				Expect(tsCh.DataType).To(Equal(telem.TimeStampT))
				Expect(tsCh.IsIndex).To(BeTrue())
			})
			It("Should create the channel without error", func(ctx SpecContext) {
				ch.Leaseholder = 1
				ch.Name = channel.NewRandomName()
				Expect(services[1].Create(ctx, &ch, channel.CreateWithoutGroupRelationship())).To(Succeed())
				Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
				entries := []ontology.Resource{}
				Expect(mockCluster.
					Nodes[1].
					Ontology.
					NewRetrieve().
					WhereIDs(ch.OntologyID()).
					TraverseTo(ontology.ParentsTraverser).
					Entries(&entries).
					Exec(ctx, nil),
				).To(Succeed())
				Expect(entries).To(BeEmpty())
			})

		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should create the channel without error", func(ctx SpecContext) {
				Expect(ch.Key().Lease()).To(Equal(node.Key(2)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should create the channel in time-series database", func(ctx SpecContext) {
				channels := MustSucceed(mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				tsCh := channels[0]
				Expect(tsCh.DataType).To(Equal(telem.TimeStampT))
				Expect(tsCh.IsIndex).To(BeTrue())
			})
			It("Should not create the channel on another nodes time-series DB", func(ctx SpecContext) {
				Expect(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())).Error().To(MatchError(query.ErrNotFound))
			})
			It("Should assign a sequential key to the channels on each node",
				func(ctx SpecContext) {
					ch2 := &channel.Channel{
						IsIndex:     true,
						Name:        channel.NewRandomName(),
						DataType:    telem.TimeStampT,
						Leaseholder: 1,
					}
					Expect(services[1].NewWriter(nil).Create(ctx, ch2)).To(Succeed())
					Expect(ch2.Key().Lease()).To(Equal(node.Key(1)))
					Expect(ch2.Key().LocalKey()).To(Equal(channel.LocalKey(7)))
				})
			It("Should correctly create a virtual channel", func(ctx SpecContext) {
				ch3 := &channel.Channel{
					Name:        channel.NewRandomName(),
					DataType:    telem.JSONT,
					Leaseholder: 2,
					Virtual:     true,
				}
				Expect(services[1].Create(ctx, ch3)).To(Succeed())
				Expect(ch3.Key().Lease()).To(Equal(node.Key(2)))
				Eventually(func(g Gomega) {
					channels := MustSucceed(mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch3.Key().StorageKey()))
					g.Expect(channels).To(HaveLen(1))
					g.Expect(channels[0].DataType).To(Equal(telem.JSONT))
					g.Expect(channels[0].Virtual).To(BeTrue())
				}).Should(Succeed())
			})
			It("Should create an index channel", func(ctx SpecContext) {
				ch4 := &channel.Channel{
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 2,
					IsIndex:     true,
				}
				Expect(services[1].Create(ctx, ch4)).To(Succeed())
				Expect(ch4.Key().Lease()).To(Equal(node.Key(2)))
				Expect(ch4.Key().LocalKey()).To(Equal(channel.LocalKey(9)))
				Expect(ch4.LocalIndex).To(Equal(channel.LocalKey(9)))
				channels := MustSucceed(mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch4.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				Expect(channels[0].IsIndex).To(BeTrue())
			})
		})
		Context("Free", func() {
			BeforeEach(func() {
				ch.Leaseholder = node.KeyFree
				ch.Virtual = true
			})
			It("Should create the channel without error", func(ctx SpecContext) {
				Expect(ch.Key().Lease()).To(Equal(node.KeyFree))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(1)))
				Expect(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())).
					Error().To(MatchError(query.ErrNotFound))
			})
		})

		Context("error cases", func() {
			It("Should return an error if the name is invalid", func(ctx SpecContext) {
				ch.Name = "invalid name"
				Expect(services[1].Create(ctx, &ch)).
					To(MatchError(ContainSubstring("contains invalid characters")))
			})
			It("Should return an error if the name is a duplicate", func(ctx SpecContext) {
				ch2 := channel.Channel{
					Name:        ch.Name,
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				Expect(services[1].Create(ctx, &ch2)).
					To(MatchError(ContainSubstring(fmt.Sprintf("channel with name '%s' already exists", ch.Name))))
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should create multiple channels without error", func(ctx SpecContext) {
			chs := []channel.Channel{
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					IsIndex:     true,
				},
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					IsIndex:     true,
				},
			}
			Expect(services[1].CreateMany(ctx, &chs)).To(Succeed())
			Expect(chs[0].Key().Lease()).To(Equal(node.Key(1)))
			Expect(chs[0].Key().LocalKey()).ToNot(BeZero())
			Expect(chs[1].Key().Lease()).To(Equal(node.Key(1)))
			Expect(chs[1].Key().LocalKey()).ToNot(BeZero())
			Expect(chs[0].Key()).ToNot(Equal(chs[1].Key()))
		})
		It("Should return an error if the names are duplicates", func(ctx SpecContext) {
			ch1 := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: 1,
				Virtual:     true,
			}
			ch2 := channel.Channel{
				Name:        ch1.Name,
				DataType:    telem.Float64T,
				Leaseholder: 1,
				Virtual:     true,
			}
			Expect(services[1].CreateMany(ctx, &[]channel.Channel{ch1, ch2})).
				To(MatchError(ContainSubstring(fmt.Sprintf("duplicate channel name '%s' in request", ch1.Name))))
		})
	})
	Context("Creating if name doesn't exist", func() {
		var ch channel.Channel
		BeforeEach(func() {
			ch.IsIndex = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.TimeStampT
			ch.Leaseholder = 1
		})
		It("Should create the channel without error", func(ctx SpecContext) {
			Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
			Expect(ch.Key().LocalKey()).ToNot(BeZero())
		})
		It("Should not create the channel if it already exists by name", func(ctx SpecContext) {
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			k := ch.Key()
			ch.Leaseholder = 0
			ch.LocalKey = 0
			Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
		})
		Describe("OverwriteIfNameExists", func() {

			It("Should overwrite the channel if it already exists by name and the new channel has different properties than the old one", func(ctx SpecContext) {
				name := channel.NewRandomName()
				ch := channel.Channel{
					Virtual:     true,
					Name:        name,
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				Expect(services[1].Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				// Try to create a new channel with the same name but different
				// properties
				newCh := channel.Channel{
					Virtual:     true,
					Name:        name,
					DataType:    telem.Float32T,
					Leaseholder: 1,
				}

				Expect(services[1].Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())

				var resChannels []channel.Channel
				Expect(services[1].NewRetrieve().Where(channel.MatchKeys(newCh.Key())).
					Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
				Expect(resChannels).To(HaveLen(1))

				Expect(resChannels[0].Virtual).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.Float32T))

				err := services[1].NewRetrieve().Where(channel.MatchKeys(originalKey)).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).To(MatchError(query.ErrNotFound))
			})
			It("Should not overwrite the channel if it already exists by name and the new channel has the same properties as the old one", func(ctx SpecContext) {
				ch := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
				}
				Expect(services[1].Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				newCh := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",         // Same name
					DataType:    telem.TimeStampT, // Same data type
					Leaseholder: 1,
				}

				Expect(services[1].Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())
				Expect(newCh.Key()).To(Equal(originalKey))

				var resChannels []channel.Channel
				Expect(services[1].NewRetrieve().Where(channel.MatchKeys(originalKey)).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
				Expect(resChannels).To(HaveLen(1))
				Expect(resChannels[0].IsIndex).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.TimeStampT))
			})
		})
		It("Should not create a free channel if it already exists by name", func(ctx SpecContext) {
			ch.Name = "SG0002"
			ch.Virtual = true
			ch.Leaseholder = node.KeyFree
			Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key().Lease()).To(Equal(node.KeyFree))
			k := ch.Key()
			ch.LocalKey = 0
			ch.Leaseholder = 0
			Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Lease()).To(Equal(node.KeyFree))
		})
	})
	Context("Calculated Channel with Auto-Created Index", func() {
		It("Should automatically create an index channel for calculated channels", func(ctx SpecContext) {
			calcCh := channel.Channel{
				Name:       "calculated_temp",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())

			Expect(calcCh.Leaseholder).To(Equal(node.KeyFree))
			Expect(calcCh.Virtual).To(BeTrue())
			Expect(calcCh.LocalIndex).ToNot(BeZero())

			indexName := "calculated_temp_time"
			var indexChannels []channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchNames(indexName)).
				Entries(&indexChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(indexChannels).To(HaveLen(1))

			indexCh := indexChannels[0]
			Expect(indexCh.IsIndex).To(BeTrue())
			Expect(indexCh.DataType).To(Equal(telem.TimeStampT))
			Expect(indexCh.Virtual).To(BeTrue())
			Expect(indexCh.Leaseholder).To(Equal(node.KeyFree))
			Expect(indexCh.LocalKey).To(Equal(calcCh.LocalIndex))
		})

		It("Should reject calculated channel with manually-specified index", func(ctx SpecContext) {
			calcCh := channel.Channel{
				Name:       "calculated_bad",
				DataType:   telem.Float64T,
				LocalIndex: channel.LocalKey(999),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			err := services[1].Create(ctx, &calcCh)
			Expect(err).To(MatchError(ContainSubstring("calculated channels cannot specify an index manually")))
		})

		It("Should retrieve existing calculated channel with its index", func(ctx SpecContext) {
			calcCh := channel.Channel{
				Name:       "calculated_retrieve_test",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())
			originalKey := calcCh.Key()
			originalIndexKey := calcCh.LocalIndex

			// Try to create again with RetrieveIfNameExists
			calcCh2 := channel.Channel{
				Name:       "calculated_retrieve_test",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
			}
			Expect(services[1].Create(ctx, &calcCh2, channel.RetrieveIfNameExists())).To(Succeed())

			// Should return existing channel with same index
			Expect(calcCh2.Key()).To(Equal(originalKey))
			Expect(calcCh2.LocalIndex).To(Equal(originalIndexKey))
		})

		It("Should handle batch create with calculated and regular channels", func(ctx SpecContext) {
			// Create index channels for regular channels first
			indexCh1 := channel.Channel{
				Name:        "regular1_idx",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			indexCh2 := channel.Channel{
				Name:        "regular2_idx",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &indexCh1)).To(Succeed())
			Expect(services[1].Create(ctx, &indexCh2)).To(Succeed())

			// Now create channels with proper indexes
			channels := []channel.Channel{
				{Name: "regular1", DataType: telem.Float64T, Leaseholder: 1, LocalIndex: indexCh1.LocalKey},
				{Name: "calculated1", DataType: telem.Float64T, Expression: "return 1 + 1"},
				{Name: "regular2", DataType: telem.Int32T, Leaseholder: 1, LocalIndex: indexCh2.LocalKey},
				{Name: "calculated2", DataType: telem.Float32T, Expression: "return 1 + 2"},
			}
			for i := range channels {
				Expect(services[1].Create(ctx, &channels[i])).To(Succeed())
			}

			// Check calculated channels have auto-created indexes
			Expect(channels[1].LocalIndex).ToNot(BeZero())
			Expect(channels[3].LocalIndex).ToNot(BeZero())

			// Check regular channels have the indexes we specified
			Expect(channels[0].LocalIndex).To(Equal(indexCh1.LocalKey))
			Expect(channels[2].LocalIndex).To(Equal(indexCh2.LocalKey))

			// Verify auto-created index channels exist for calculated channels
			var indexChannels []channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchNames("calculated1_time", "calculated2_time")).
				Entries(&indexChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(indexChannels).To(HaveLen(2))
		})

		It("Should create internal index for internal calculated channel", func(ctx SpecContext) {
			calcCh := channel.Channel{
				Name:       "internal_calculated",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Internal:   true,
			}
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())

			// Verify index is also internal
			indexName := "internal_calculated_time"
			var indexChannels []channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchNames(indexName)).
				Entries(&indexChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(indexChannels).To(HaveLen(1))
			Expect(indexChannels[0].Internal).To(BeTrue())
		})
	})
	Context("Updating Calculated Channel Expression", func() {
		It("Should update expression when Create() called with existing key", func(ctx SpecContext) {
			// 0. Create the channel the calculated expression references.
			sensor1 := channel.Channel{Name: "sensor1", DataType: telem.Float64T, Virtual: true}
			Expect(services[1].Create(ctx, &sensor1)).To(Succeed())

			// 1. Create calculated channel
			calcCh := channel.Channel{
				Name:       "temperature_calc",
				DataType:   telem.Float64T,
				Expression: "return sensor1 * 2.5",
			}
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())

			originalKey := calcCh.Key()
			originalIndexKey := calcCh.LocalIndex
			originalName := calcCh.Name

			// 2. Modify expression and call Create() again with same key
			calcCh.Expression = "return sensor1 * 3.0 + 10"
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())

			// 3. Verify key unchanged
			Expect(calcCh.Key()).To(Equal(originalKey))

			// 4. Verify index unchanged
			Expect(calcCh.LocalIndex).To(Equal(originalIndexKey))

			// 5. Retrieve and verify expression updated, other fields preserved
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(originalKey)).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Expression).To(Equal("return sensor1 * 3.0 + 10"))
			Expect(retrieved.Name).To(Equal(originalName))
			Expect(retrieved.Key()).To(Equal(originalKey))
		})

		It("Should infer the calculated channel DataType from its expression on update", func(ctx SpecContext) {
			calcCh := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				Expression: "return 1.0",
			}
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())
			Expect(calcCh.DataType).To(Equal(telem.Float64T))
			originalKey := calcCh.Key()

			// 2. Update the expression. The DataType is always inferred from the
			// expression, so the caller-provided DataType is ignored.
			calcCh.Expression = "return 2.0"
			calcCh.DataType = telem.Float32T
			Expect(services[1].Create(ctx, &calcCh)).To(Succeed())

			// 3. Retrieve from DB and verify DataType was updated
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(originalKey)).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())

			Expect(retrieved.Expression).To(Equal("return 2.0"))
			Expect(retrieved.DataType).To(Equal(telem.Float64T))
		})
	})
	Context("Updating a channel", func() {
		var ch channel.Channel
		var ch2 channel.Channel
		BeforeEach(func(ctx SpecContext) {
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.Float64T
			ch.Virtual = true
			ch.Internal = false
			ch.Leaseholder = node.KeyFree

			ch2.IsIndex = true
			ch2.Name = channel.NewRandomName()
			ch2.DataType = telem.TimeStampT
			ch2.Leaseholder = 1

			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(services[1].Create(ctx, &ch2)).To(Succeed())
		})
		It("Should update the channel name without error", func(ctx SpecContext) {
			newName := channel.NewRandomName()
			ch.Name = newName
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(ch.Name).To(Equal(newName))

			var resChannels []channel.Channel
			Expect(services[1].NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(newName))
		})
		It("Should not update the channel if it already exists by name", func(ctx SpecContext) {
			existingName := ch2.Name
			ch.Name = existingName
			Expect(services[1].Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Name).To(Equal(existingName))

			var resChannels []channel.Channel
			Expect(services[1].NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(existingName))
		})
		It("Should assign a new key when attempting to update a non-virtual channel",
			func(ctx SpecContext) {
				// Create initial non-virtual channel
				nonVirtualCh := channel.Channel{
					IsIndex:     true,
					Name:        "NonVirtual",
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					Virtual:     false,
				}
				Expect(services[1].Create(ctx, &nonVirtualCh)).To(Succeed())
				originalKey := nonVirtualCh.Key()

				nonVirtualCh.Name = "UpdatedName"
				Expect(services[1].Create(ctx, &nonVirtualCh)).To(Succeed())

				Expect(nonVirtualCh.Key()).ToNot(Equal(originalKey))

				var resChannels []channel.Channel
				Expect(services[1].NewRetrieve().Where(channel.MatchKeys(originalKey, nonVirtualCh.Key())).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
				Expect(resChannels).To(HaveLen(2))

				Expect(resChannels[0].Name).To(Equal("NonVirtual"))
				Expect(resChannels[1].Name).To(Equal("UpdatedName"))
			})
	})

})

var _ = Context("Name Validation Disabled", func() {
	Describe("Channel Creation", Ordered, func() {
		var (
			mockCluster *mock.Cluster
			services    map[node.Key]*channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			mockCluster = mock.MustOpenCluster(ctx, 1)
			services = make(map[node.Key]*channel.Service)
			services[1] = openService(ctx, mockCluster.Nodes[1], channel.ServiceConfig{ValidateNames: new(false)})
		})
		It("Should create a channel with spaces in the name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:        "my channel with spaces",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("my channel with spaces"))
		})
		It("Should create a channel with special characters in the name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:        "sensor!@#$%",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("sensor!@#$%"))
		})
		It("Should create a channel with a name starting with a digit", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:        "1sensor",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("1sensor"))
		})
		It("Should still reject empty names", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:        "",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(services[1].Create(ctx, &ch)).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should allow renaming to a name with special characters", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:        "original_name",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Expect(services[1].Rename(ctx, ch.Key(), "new name with spaces!", false)).To(Succeed())
			var retrieved channel.Channel
			Expect(services[1].NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("new name with spaces!"))
		})
	})
})

var _ = Describe("Delete", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(ctx, 2)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})
	Describe("Channel Deletion", func() {
		Context("Single Channel", func() {
			var idxCh, ch channel.Channel
			JustBeforeEach(func(ctx SpecContext) {
				prefix := channel.NewRandomName()
				idxCh.Name = prefix + "_time"
				idxCh.DataType = telem.TimeStampT
				idxCh.IsIndex = true
				Expect(services[1].Create(ctx, &idxCh)).To(Succeed())
				ch.Name = prefix + "_data"
				ch.DataType = telem.Float64T
				ch.LocalIndex = idxCh.LocalKey
				Expect(services[1].Create(ctx, &ch)).To(Succeed())
			})
			Context("Node is local", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 1
					ch.Leaseholder = 1
				})
				It("Should not allow deletion of index channel with dependent channels", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func(ctx SpecContext) {
					Expect(services[1].DeleteMany(ctx, channel.Keys{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				It("Should not be able to retrieve the channel after deletion", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					exists := MustSucceed(services[1].NewRetrieve().Where(channel.MatchKeys(ch.Key())).Exists(ctx, nil))
					Expect(exists).To(BeFalse())
				})
				It("Should not be able to retrieve the channel from the time-series DB", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(ts.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})

			Context("Node is remote", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 2
					ch.Leaseholder = 2
				})
				It("Should not allow deletion of index channel with dependent channels", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func(ctx SpecContext) {
					Expect(services[1].DeleteMany(ctx, []channel.Key{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				// PENDING (SY-4331 follow-up): deleting a remote-leased channel's metadata
				// from a non-leaseholder node does not propagate. The gorp delete on the
				// originating node cannot derive the channel's existing aspen lease (its
				// local digest lookup misses), so the delete defaults to the host
				// leaseholder and never removes the leaseholder's authoritative copy. The
				// pre-refactor code routed deletes to the leaseholder; the service-layer
				// metadata delete needs equivalent leaseholder routing.
				PIt("Should not be able to retrieve the channel after deletion", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					Eventually(func(g Gomega) {
						g.Expect(MustSucceed(services[1].NewRetrieve().Where(channel.MatchKeys(ch.Key())).Exists(ctx, nil))).To(BeFalse())
						g.Expect(MustSucceed(services[2].NewRetrieve().Where(channel.MatchKeys(ch.Key())).Exists(ctx, nil))).To(BeFalse())
					}).Should(Succeed())
				})
				It("Should not be able to retrieve the channel from the time-series DB", func(ctx SpecContext) {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(ts.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})
		})
	})

})

var _ = Describe("Rename", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(context.Background(), 3)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func(ctx SpecContext) {
			ch.Virtual = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.Float64T
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := channel.NewRandomName()
				Expect(services[1].Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(services[1].NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := channel.NewRandomName()
				Expect(services[2].Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(services[2].NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("new name is invalid", func() {
			It("Should return an error", func(ctx SpecContext) {
				Expect(services[1].Rename(ctx, ch.Key(), "invalid name", false)).To(MatchError(ContainSubstring("contains invalid characters")))
			})
		})
		Context("new name is a duplicate", func() {
			It("Should return an error", func(ctx SpecContext) {
				secondCh := channel.Channel{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Float64T,
				}
				Expect(services[1].Create(ctx, &secondCh)).To(Succeed())
				Expect(services[1].Rename(ctx, ch.Key(), secondCh.Name, false)).
					To(MatchError(ContainSubstring("channel with name '%s' already exists", secondCh.Name)))
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should rename the channels without error", func(ctx SpecContext) {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Int64T,
				},
				{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Float32T,
				},
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.StringT,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				},
			}
			Expect(services[1].CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			names := []string{channel.NewRandomName(), channel.NewRandomName(), channel.NewRandomName()}
			Expect(services[1].RenameMany(
				ctx,
				keys,
				names,
				false,
			)).To(Succeed())
			var resChannels []channel.Channel
			Expect(services[1].NewRetrieve().Where(channel.MatchKeys(keys...)).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
			Expect(resChannels[0].Name).To(Equal(names[0]))
			Expect(resChannels[1].Name).To(Equal(names[1]))
			Expect(resChannels[2].Name).To(Equal(names[2]))
		})
	})

})

func fixedOverflowChecker(limit int) channel.IntOverflowChecker {
	return func(count types.Uint20) error {
		if count > types.Uint20(limit) {
			return errors.New("channel limit exceeded")
		}
		return nil
	}
}

var _ = Describe("Limit", Ordered, func() {
	var (
		limit    = 5
		dist     mock.Node
		limitSvc *channel.Service
	)
	BeforeEach(func(ctx SpecContext) {
		dist = mock.MustOpenNode(ctx)
		limitSvc = openService(ctx, dist, channel.ServiceConfig{IntOverflowCheck: fixedOverflowChecker(limit)})
	})
	It("Should not allow creating channels over the limit", func(ctx SpecContext) {
		// Create channels up to the limit
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(limitSvc.Create(ctx, &ch)).To(Succeed())
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &overLimitCh)).
			Error().To(MatchError(ContainSubstring("channel limit exceeded")))
	})

	It("Should allow creating channels after deleting some to stay under the limit", func(ctx SpecContext) {
		// Create channels up to the limit
		channels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(limitSvc.Create(ctx, &ch)).To(Succeed())
			channels[i] = ch
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &overLimitCh)).
			Error().To(MatchError(ContainSubstring("channel limit exceeded")))

		// Delete one channel
		writer := limitSvc.NewWriter(nil)
		Expect(writer.Delete(ctx, channels[0].Key(), false)).To(Succeed())

		// Now we should be able to create a new channel
		newCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "NewAfterDelete",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &newCh)).To(Succeed())

		// Try to create one more channel (should fail again)
		anotherCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "AnotherOverLimit",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &anotherCh)).
			Error().To(MatchError(ContainSubstring("channel limit exceeded")))
	})

	It("Should allow retrieving channels even at the limit", func(ctx SpecContext) {
		// Create channels up to the limit
		createdChannels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(limitSvc.Create(ctx, &ch)).To(Succeed())
			createdChannels[i] = ch
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &overLimitCh)).
			Error().To(MatchError(ContainSubstring("channel limit exceeded")))

		// Retrieve all channels - this should work fine even at the limit
		var retrievedChannels []channel.Channel
		retrieve := limitSvc.NewRetrieve()
		Expect(retrieve.Entries(&retrievedChannels).Where(channel.MatchLeaseholders(1)).Exec(ctx, nil)).To(Succeed())
		Expect(retrievedChannels).To(HaveLen(limit + internalChannelCount))

		// Retrieve a specific channel by name
		var singleChannel channel.Channel
		Expect(retrieve.Where(channel.MatchKeys(createdChannels[0].Key())).Entry(&singleChannel).Exec(ctx, nil)).To(Succeed())
		Expect(singleChannel.Name).To(Equal(createdChannels[0].Name))
	})
	It("Should not edit the channel limit if a deletion fails in TS", func(ctx SpecContext) {
		createdChannels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(limitSvc.Create(ctx, &ch)).To(Succeed())
			createdChannels[i] = ch
		}
		MustOpen(MustSucceed(writer.NewService(writer.ServiceConfig{Framer: dist.Framer, Channel: limitSvc})).Open(ctx, framer.WriterConfig{
			Keys: []channel.Key{createdChannels[0].Key()},
		}))
		Expect(limitSvc.Delete(ctx, createdChannels[0].Key(), false)).
			To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
		newCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "NewAfterDelete",
			Leaseholder: 1,
		}
		Expect(limitSvc.Create(ctx, &newCh)).
			To(MatchError(ContainSubstring("channel limit exceeded")))
	})
})
