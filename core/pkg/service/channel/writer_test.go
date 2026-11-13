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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	xfstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

func fixedLimit(limit int) channel.LimitChecker {
	return func(count types.Uint20) error {
		if count > types.Uint20(limit) {
			return errors.New("channel limit exceeded")
		}
		return nil
	}
}

var _ = Describe("Writer", func() {
	// The external non-virtual channel set is not observable, so these specs exercise
	// it through the channel limit: only external non-virtual channels count toward it.
	Describe("External Channel Limit", Ordered, func() {
		var (
			db       *gorp.DB
			limitSvc *channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist := testutil.NewNode(ctx)
			db = dist.DB
			limitSvc = openService(ctx, dist, channel.ServiceConfig{
				Limit: fixedLimit(2),
			})
		})
		It(
			"Should not count virtual or internal channels toward the limit",
			func(ctx SpecContext) {
				for range 5 {
					virtual := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.Float64T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &virtual)
					})).To(Succeed())
					internal := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Internal:    true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &internal)
					})).To(Succeed())
				}
			},
		)
		It(
			"Should reject external non-virtual channels past the limit",
			func(ctx SpecContext) {
				for range 2 {
					ch := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
				}
				third := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &third)
				})).
					Error().To(MatchError(ContainSubstring("channel limit exceeded")))
			},
		)
		It(
			"Should remove overwritten channels from the external channel set",
			func(ctx SpecContext) {
				cleanup := testutil.NewNode(ctx)
				cleanupSvc := openService(
					ctx,
					cleanup,
					channel.ServiceConfig{
						Limit: fixedLimit(3),
					},
				)
				idx := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: 1,
				}
				Expect(cleanup.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return cleanupSvc.NewWriter(tx).Create(ctx, &idx)
				})).To(Succeed())
				name := UniqueChannelName()
				data := channel.Channel{
					Name:        name,
					DataType:    telem.Float64T,
					LocalIndex:  idx.LocalKey,
					Leaseholder: 1,
				}
				Expect(cleanup.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return cleanupSvc.NewWriter(tx).Create(ctx, &data)
				})).To(Succeed())

				overwrite := channel.Channel{
					Name:        name,
					DataType:    telem.Float32T,
					LocalIndex:  idx.LocalKey,
					Leaseholder: 1,
				}
				Expect(cleanup.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return cleanupSvc.NewWriter(tx).Create(ctx, &overwrite,
						channel.OverwriteIfNameExistsAndDifferentProperties(),
					)
				})).To(Succeed())

				// The set now holds {idx, overwrite}. If the overwritten channel's key
				// had been left behind, creating another external channel would push
				// the count to 4 and trip the limit of 3.
				another := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float64T,
					LocalIndex:  idx.LocalKey,
					Leaseholder: 1,
				}
				Expect(cleanup.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return cleanupSvc.NewWriter(tx).Create(ctx, &another)
				})).To(Succeed())
			},
		)
		Describe("Rolled back transaction", func() {
			var (
				oneSvc   *channel.Service
				oneDB    *gorp.DB
				rollback = errors.New("rollback")
				create   = func(ctx SpecContext, ch *channel.Channel) error {
					return oneDB.WithTx(ctx, func(tx gorp.Tx) error {
						return oneSvc.NewWriter(tx).Create(ctx, ch)
					})
				}
			)
			BeforeEach(func(ctx SpecContext) {
				one := testutil.NewNode(ctx)
				oneDB = one.DB
				oneSvc = openService(
					ctx,
					one,
					channel.ServiceConfig{Limit: fixedLimit(1)},
				)
			})
			It("Should not count a create that rolls back", func(ctx SpecContext) {
				Expect(oneDB.WithTx(ctx, func(tx gorp.Tx) error {
					ch := channel.Channel{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
					}
					if err := oneSvc.NewWriter(tx).Create(ctx, &ch); err != nil {
						return err
					}
					return rollback
				})).To(MatchError(rollback))
				Expect(create(ctx, &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				})).To(Succeed())
			})
			It(
				"Should keep counting a channel whose delete rolls back",
				func(ctx SpecContext) {
					ch := channel.Channel{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
					}
					Expect(create(ctx, &ch)).To(Succeed())
					Expect(oneDB.WithTx(ctx, func(tx gorp.Tx) error {
						if err := oneSvc.NewWriter(tx).
							Delete(ctx, ch.Key(), false); err != nil {
							return err
						}
						return rollback
					})).To(MatchError(rollback))
					Expect(create(ctx, &channel.Channel{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
					})).To(MatchError(ContainSubstring("channel limit exceeded")))
				},
			)
		})
	})

	// Name validation is enforced internally on the create/rename path, so these specs
	// exercise it through svc.Create rather than calling the unexported validator
	// directly.
	Describe("Name Validation", func() {
		DescribeTable("Should accept valid names", func(ctx SpecContext, name string) {
			ch := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
		},
			Entry("only letters", "name_valid_temperature"),
			Entry("mixed case", "name_valid_Pressure"),
			Entry("with digits", "name_valid_sensor1"),
			Entry("with underscores", "name_valid_sensor_temp"),
			Entry("letters, digits, and underscores", "name_valid_temp123_sensor"),
		)
		DescribeTable(
			"Should reject invalid names",
			func(ctx SpecContext, name, msg string) {
				ch := channel.Channel{
					Name:     name,
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("empty name", "", "name: required"),
			Entry("name starting with a digit", "1sensor", "cannot start with a digit"),
			Entry("name with spaces", "my channel", "contains invalid characters"),
			Entry(
				"name with special characters",
				"sensor!",
				"contains invalid characters",
			),
		)
	})

	Describe("Create", Ordered, func() {
		var (
			dist testutil.Node
			db   *gorp.DB
			svc  *channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist = testutil.NewNode(ctx)
			db = dist.DB
			svc = openService(ctx, dist)
		})
		Context("Single channel", func() {
			var ch channel.Channel
			JustBeforeEach(func(ctx SpecContext) {
				ch.IsIndex = true
				ch.Name = UniqueChannelName()
				ch.DataType = telem.TimestampT
				ch.Leaseholder = 1
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
			})
			Context("error cases", func() {
				It(
					"Should return an error if the name is invalid",
					func(ctx SpecContext) {
						ch.Name = "invalid name"
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch)
						})).
							To(MatchError(ContainSubstring("contains invalid characters")))
					},
				)
				It(
					"Should return an error if the name is a duplicate",
					func(ctx SpecContext) {
						ch2 := channel.Channel{
							Name:        ch.Name,
							DataType:    telem.Float64T,
							Leaseholder: 1,
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch2)
						})).
							To(MatchError(ContainSubstring(fmt.Sprintf("channel with name '%s' already exists", ch.Name))))
					},
				)
				It(
					"Should return a validation error if an operation has an invalid type",
					func(ctx SpecContext) {
						ch2 := channel.Channel{
							Name:        UniqueChannelName(),
							DataType:    telem.Float64T,
							Leaseholder: 1,
							Operations:  []channel.Operation{{Type: "not-a-real-op"}},
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch2)
						})).
							To(MatchError(ContainSubstring("operations.0.type: invalid type")))
					},
				)
			})
		})
		Context("Multiple channels", func() {
			It("Should create multiple channels without error", func(ctx SpecContext) {
				chs := []channel.Channel{
					{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						Leaseholder: 1,
						IsIndex:     true,
					},
					{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						Leaseholder: 1,
						IsIndex:     true,
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).CreateMany(ctx, &chs)
				})).To(Succeed())
				Expect(chs[0].Key().Lease()).To(Equal(node.Key(1)))
				Expect(chs[0].Key().LocalKey()).ToNot(BeZero())
				Expect(chs[1].Key().Lease()).To(Equal(node.Key(1)))
				Expect(chs[1].Key().LocalKey()).ToNot(BeZero())
				Expect(chs[0].Key()).ToNot(Equal(chs[1].Key()))
			})
			It(
				"Should return an error if the names are duplicates",
				func(ctx SpecContext) {
					ch1 := channel.Channel{
						Name:        UniqueChannelName(),
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
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							CreateMany(ctx, &[]channel.Channel{ch1, ch2})
					})).
						To(MatchError(ContainSubstring(fmt.Sprintf("duplicate channel name '%s' in request", ch1.Name))))
				},
			)
		})
		Context("Creating if name doesn't exist", func() {
			var ch channel.Channel
			BeforeEach(func() {
				ch.IsIndex = true
				ch.Name = UniqueChannelName()
				ch.DataType = telem.TimestampT
				ch.Leaseholder = 1
			})
			It("Should create the channel without error", func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						Create(ctx, &ch, channel.RetrieveIfNameExists())
				})).To(Succeed())
				Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
				Expect(ch.Key().LocalKey()).ToNot(BeZero())
			})
			It(
				"Should not create the channel if it already exists by name",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					k := ch.Key()
					ch.Leaseholder = 0
					ch.LocalKey = 0
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Create(ctx, &ch, channel.RetrieveIfNameExists())
					})).To(Succeed())
					Expect(ch.Key()).To(Equal(k))
					Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
				},
			)
			Describe("OverwriteIfNameExists", func() {
				It(
					"Should overwrite the channel if it already exists by name and the new channel has different properties than the old one",
					func(ctx SpecContext) {
						name := UniqueChannelName()
						ch := channel.Channel{
							Virtual:     true,
							Name:        name,
							DataType:    telem.Float64T,
							Leaseholder: 1,
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch)
						})).To(Succeed())
						originalKey := ch.Key()

						// Try to create a new channel with the same name but different
						// properties
						newCh := channel.Channel{
							Virtual:     true,
							Name:        name,
							DataType:    telem.Float32T,
							Leaseholder: 1,
						}

						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(
								ctx,
								&newCh,
								channel.OverwriteIfNameExistsAndDifferentProperties(),
							)
						})).To(Succeed())

						var resChannels []channel.Channel
						Expect(svc.NewRetrieve().Where(channel.MatchKeys(newCh.Key())).
							Entries(&resChannels).Exec(ctx, db)).To(Succeed())
						Expect(resChannels).To(HaveLen(1))

						Expect(resChannels[0].Virtual).To(BeTrue())
						Expect(resChannels[0].DataType).To(Equal(telem.Float32T))

						Expect(
							svc.NewRetrieve().
								Where(channel.MatchKeys(originalKey)).
								Entries(&resChannels).
								Exec(ctx, db),
						).To(MatchError(query.ErrNotFound))
					},
				)
				It(
					"Should not overwrite the channel if it already exists by name and the new channel has the same properties as the old one",
					func(ctx SpecContext) {
						ch := channel.Channel{
							IsIndex:     true,
							Name:        "SG0001",
							DataType:    telem.TimestampT,
							Leaseholder: 1,
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch)
						})).To(Succeed())
						originalKey := ch.Key()

						newCh := channel.Channel{
							IsIndex:     true,
							Name:        "SG0001",         // Same name
							DataType:    telem.TimestampT, // Same data type
							Leaseholder: 1,
						}

						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(
								ctx,
								&newCh,
								channel.OverwriteIfNameExistsAndDifferentProperties(),
							)
						})).To(Succeed())
						Expect(newCh.Key()).To(Equal(originalKey))

						var resChannels []channel.Channel
						Expect(
							svc.NewRetrieve().
								Where(channel.MatchKeys(originalKey)).
								Entries(&resChannels).
								Exec(ctx, db),
						).To(Succeed())
						Expect(resChannels).To(HaveLen(1))
						Expect(resChannels[0].IsIndex).To(BeTrue())
						Expect(resChannels[0].DataType).To(Equal(telem.TimestampT))
					},
				)
				It(
					"Should leave the overwritten channel out of its resources",
					func(ctx SpecContext) {
						name := UniqueChannelName()
						ch := channel.Channel{
							Virtual:     true,
							Name:        name,
							DataType:    telem.Float64T,
							Leaseholder: 1,
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &ch)
						})).To(Succeed())
						originalKey := ch.Key()

						newCh := channel.Channel{
							Virtual:     true,
							Name:        name,
							DataType:    telem.Float32T,
							Leaseholder: 1,
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(
								ctx,
								&newCh,
								channel.OverwriteIfNameExistsAndDifferentProperties(),
							)
						})).To(Succeed())
						Expect(newCh.Key()).ToNot(Equal(originalKey))

						Expect(svc.Resources().Retrieve(ctx, dist.DB, []resource.ID{
							channel.ResourceID(originalKey),
							channel.ResourceID(newCh.Key()),
						})).To(ConsistOf(resource.Resource{
							ID:   channel.ResourceID(newCh.Key()),
							Name: name,
						}))
					},
				)
			})
			It(
				"Should not create a free channel if it already exists by name",
				func(ctx SpecContext) {
					ch.Name = "SG0002"
					ch.Virtual = true
					ch.Leaseholder = node.KeyFree
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Create(ctx, &ch, channel.RetrieveIfNameExists())
					})).To(Succeed())
					Expect(ch.Key().Lease()).To(Equal(node.KeyFree))
					k := ch.Key()
					ch.LocalKey = 0
					ch.Leaseholder = 0
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Create(ctx, &ch, channel.RetrieveIfNameExists())
					})).To(Succeed())
					Expect(ch.Key()).To(Equal(k))
					Expect(ch.Key().Lease()).To(Equal(node.KeyFree))
				},
			)
		})
		Context("Calculated Channel with Auto-Created Index", func() {
			It(
				"Should automatically create an index channel for calculated channels",
				func(ctx SpecContext) {
					calcCh := channel.Channel{
						Name:       "calculated_temp",
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
						Virtual:    true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(Succeed())

					Expect(calcCh.Leaseholder).To(Equal(node.KeyFree))
					Expect(calcCh.Virtual).To(BeTrue())
					Expect(calcCh.LocalIndex).ToNot(BeZero())

					indexName := "calculated_temp_time"
					var indexChannels []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(indexName)).
						Entries(&indexChannels).
						Exec(ctx, db)).To(Succeed())
					Expect(indexChannels).To(HaveLen(1))

					indexCh := indexChannels[0]
					Expect(indexCh.IsIndex).To(BeTrue())
					Expect(indexCh.DataType).To(Equal(telem.TimestampT))
					Expect(indexCh.Virtual).To(BeTrue())
					Expect(indexCh.Leaseholder).To(Equal(node.KeyFree))
					Expect(indexCh.LocalKey).To(Equal(calcCh.LocalIndex))
				},
			)

			It(
				"Should reject calculated channel with manually-specified index",
				func(ctx SpecContext) {
					calcCh := channel.Channel{
						Name:       "calculated_bad",
						DataType:   telem.Float64T,
						LocalIndex: channel.LocalKey(999),
						Expression: "return 1 + 1",
						Virtual:    true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(
						MatchError(
							ContainSubstring(
								"calculated channels cannot specify an index manually",
							),
						),
					)
				},
			)

			It(
				"Should retrieve existing calculated channel with its index",
				func(ctx SpecContext) {
					calcCh := channel.Channel{
						Name:       "calculated_retrieve_test",
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
						Virtual:    true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(Succeed())
					originalKey := calcCh.Key()
					originalIndexKey := calcCh.LocalIndex

					// Try to create again with RetrieveIfNameExists
					calcCh2 := channel.Channel{
						Name:       "calculated_retrieve_test",
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(
							ctx,
							&calcCh2,
							channel.RetrieveIfNameExists(),
						)
					})).To(Succeed())

					// Should return existing channel with same index
					Expect(calcCh2.Key()).To(Equal(originalKey))
					Expect(calcCh2.LocalIndex).To(Equal(originalIndexKey))
				},
			)

			It(
				"Should handle batch create with calculated and regular channels",
				func(ctx SpecContext) {
					// Create index channels for regular channels first
					indexCh1 := channel.Channel{
						Name:        "regular1_idx",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					indexCh2 := channel.Channel{
						Name:        "regular2_idx",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &indexCh1)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &indexCh2)
					})).To(Succeed())

					// Now create channels with proper indexes
					channels := []channel.Channel{
						{
							Name:        "regular1",
							DataType:    telem.Float64T,
							Leaseholder: 1,
							LocalIndex:  indexCh1.LocalKey,
						},
						{
							Name:       "calculated1",
							DataType:   telem.Float64T,
							Expression: "return 1 + 1",
						},
						{
							Name:        "regular2",
							DataType:    telem.Int32T,
							Leaseholder: 1,
							LocalIndex:  indexCh2.LocalKey,
						},
						{
							Name:       "calculated2",
							DataType:   telem.Float32T,
							Expression: "return 1 + 2",
						},
					}
					for i := range channels {
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Create(ctx, &channels[i])
						})).To(Succeed())
					}

					// Check calculated channels have auto-created indexes
					Expect(channels[1].LocalIndex).ToNot(BeZero())
					Expect(channels[3].LocalIndex).ToNot(BeZero())

					// Check regular channels have the indexes we specified
					Expect(channels[0].LocalIndex).To(Equal(indexCh1.LocalKey))
					Expect(channels[2].LocalIndex).To(Equal(indexCh2.LocalKey))

					// Verify auto-created index channels exist for calculated channels
					var indexChannels []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames("calculated1_time", "calculated2_time")).
						Entries(&indexChannels).
						Exec(ctx, db)).To(Succeed())
					Expect(indexChannels).To(HaveLen(2))
				},
			)

			It(
				"Should reject a calculated channel whose auto-created index name collides with an existing channel",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					taken := channel.Channel{
						Name:     name + "_time",
						DataType: telem.Float64T,
						Virtual:  true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &taken)
					})).To(Succeed())
					calcCh := channel.Channel{
						Name:       name,
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"cannot serve as the index for a calculated channel",
							),
						),
					))
				},
			)

			It(
				"Should adopt an orphaned index when creating a calculated channel",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					// A Core older than 0.61 left the index of a deleted calculated
					// channel behind.
					orphan := channel.Channel{
						Name:        name + "_time",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Virtual:     true,
						Leaseholder: node.KeyFree,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &orphan)
					})).To(Succeed())
					originalIndex := orphan.LocalKey
					recreated := channel.Channel{
						Name:       name,
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &recreated)
					})).To(Succeed())
					Expect(recreated.LocalIndex).To(Equal(originalIndex))

					var indexChannels []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(name+"_time")).
						Entries(&indexChannels).
						Exec(ctx, db)).To(Succeed())
					Expect(indexChannels).To(HaveLen(1))
				},
			)

			It(
				"Should reject adopting a non-index channel as a calculated channel's index",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					taken := channel.Channel{
						Name:     name + "_time",
						DataType: telem.Float64T,
						Virtual:  true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &taken)
					})).To(Succeed())
					calcCh := channel.Channel{
						Name:       name,
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(
							ctx,
							&calcCh,
							channel.RetrieveIfNameExists(),
						)
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"cannot serve as the index for a calculated channel",
							),
						),
					))
				},
			)

			It(
				"Should reject a batch containing a non-index channel that collides with an auto-created index name",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					chs := []channel.Channel{
						{
							Name:       name,
							DataType:   telem.Float64T,
							Expression: "return 1 + 1",
						},
						{Name: name + "_time", DataType: telem.Float64T, Virtual: true},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).CreateMany(ctx, &chs)
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"cannot serve as the index for calculated channel",
							),
						),
					))
				},
			)

			It(
				"Should reject a batch containing a persisted index channel that collides with an auto-created index name",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					chs := []channel.Channel{
						{
							Name:       name,
							DataType:   telem.Float64T,
							Expression: "return 1 + 1",
						},
						{
							Name:     name + "_time",
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).CreateMany(ctx, &chs)
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"cannot serve as the index for calculated channel",
							),
						),
					))
				},
			)

			It(
				"Should adopt a virtual free index channel in the same batch as a calculated channel",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					chs := []channel.Channel{
						{
							Name:       name,
							DataType:   telem.Float64T,
							Expression: "return 1 + 1",
						},
						{
							Name:        name + "_time",
							DataType:    telem.TimestampT,
							IsIndex:     true,
							Virtual:     true,
							Leaseholder: node.KeyFree,
						},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).CreateMany(ctx, &chs)
					})).To(Succeed())
					Expect(chs[0].LocalIndex).To(Equal(chs[1].LocalKey))
					Expect(chs[0].Index()).To(Equal(chs[1].Key()))
				},
			)

			It(
				"Should reuse the existing index when overwriting a calculated channel",
				func(ctx SpecContext) {
					name := UniqueChannelName()
					calcCh := channel.Channel{
						Name:       name,
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(Succeed())
					originalIndex := calcCh.LocalIndex

					replacement := channel.Channel{
						Name:       name,
						DataType:   telem.Float64T,
						Expression: "return 2 + 2",
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(
							ctx,
							&replacement,
							channel.OverwriteIfNameExistsAndDifferentProperties(),
						)
					})).To(Succeed())

					var indexChannels []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(name+"_time")).
						Entries(&indexChannels).
						Exec(ctx, db)).To(Succeed())
					Expect(indexChannels).To(HaveLen(1))
					Expect(indexChannels[0].LocalKey).To(Equal(originalIndex))
					Expect(replacement.LocalIndex).To(Equal(originalIndex))
				},
			)

			It(
				"Should create internal index for internal calculated channel",
				func(ctx SpecContext) {
					calcCh := channel.Channel{
						Name:       "internal_calculated",
						DataType:   telem.Float64T,
						Expression: "return 1 + 1",
						Internal:   true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &calcCh)
					})).To(Succeed())

					// Verify index is also internal
					indexName := "internal_calculated_time"
					var indexChannels []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(indexName)).
						Entries(&indexChannels).
						Exec(ctx, db)).To(Succeed())
					Expect(indexChannels).To(HaveLen(1))
					Expect(indexChannels[0].Internal).To(BeTrue())
				},
			)
		})
		Context("Calculated channel with an existing key", func() {
			var calc channel.Channel
			BeforeEach(func(ctx SpecContext) {
				calc = channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float64T,
					Expression: "return 1.0",
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &calc)
				})).To(Succeed())
			})
			It("Should refuse a channel whose key exists", func(ctx SpecContext) {
				again := calc
				again.Name = UniqueChannelName()
				again.Expression = "return 2.0"
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &again)
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("channel %s already exists", calc.Key()),
					),
				))
				var stored channel.Channel
				Expect(svc.NewRetrieve().
					Where(channel.MatchKeys(calc.Key())).
					Entry(&stored).
					Exec(ctx, db)).To(Succeed())
				Expect(stored.Name).To(Equal(calc.Name))
				Expect(stored.Expression).To(Equal("return 1.0"))
			})
			It(
				"Should return the stored channel with RetrieveIfNameExists",
				func(ctx SpecContext) {
					again := calc
					again.Name = UniqueChannelName()
					again.Expression = "return 2.0"
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Create(ctx, &again, channel.RetrieveIfNameExists())
					})).To(Succeed())
					Expect(again.Name).To(Equal(calc.Name))
					Expect(again.Expression).To(Equal("return 1.0"))
					var stored channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(calc.Key())).
						Entry(&stored).
						Exec(ctx, db)).To(Succeed())
					Expect(stored.Name).To(Equal(calc.Name))
					Expect(stored.Expression).To(Equal("return 1.0"))
				},
			)
		})
		Context("Updating a channel", func() {
			var ch channel.Channel
			var ch2 channel.Channel
			BeforeEach(func(ctx SpecContext) {
				ch = channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
				}
				ch2 = channel.Channel{
					IsIndex:     true,
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch2)
				})).To(Succeed())
			})
			It("Should update the channel name without error", func(ctx SpecContext) {
				newName := UniqueChannelName()
				ch.Name = newName
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				Expect(ch.Name).To(Equal(newName))

				var resChannels []channel.Channel
				Expect(
					svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entries(&resChannels).
						Exec(ctx, db),
				).To(Succeed())
				Expect(resChannels).To(HaveLen(1))
				Expect(resChannels[0].Name).To(Equal(newName))
			})
			It(
				"Should not update the channel if it already exists by name",
				func(ctx SpecContext) {
					existingName := ch2.Name
					ch.Name = existingName
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Create(ctx, &ch, channel.RetrieveIfNameExists())
					})).To(Succeed())
					Expect(ch.Name).To(Equal(existingName))

					var resChannels []channel.Channel
					Expect(
						svc.NewRetrieve().
							Where(channel.MatchKeys(ch.Key())).
							Entries(&resChannels).
							Exec(ctx, db),
					).To(Succeed())
					Expect(resChannels).To(HaveLen(1))
					Expect(resChannels[0].Name).To(Equal(existingName))
				},
			)
			It(
				"Should assign a new key when attempting to update a non-virtual channel",
				func(ctx SpecContext) {
					// Create initial non-virtual channel
					nonVirtualCh := channel.Channel{
						IsIndex:     true,
						Name:        "NonVirtual",
						DataType:    telem.TimestampT,
						Leaseholder: 1,
						Virtual:     false,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &nonVirtualCh)
					})).To(Succeed())
					originalKey := nonVirtualCh.Key()

					nonVirtualCh.Name = "UpdatedName"
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &nonVirtualCh)
					})).To(Succeed())

					Expect(nonVirtualCh.Key()).ToNot(Equal(originalKey))

					var resChannels []channel.Channel
					Expect(
						svc.NewRetrieve().
							Where(channel.MatchKeys(originalKey, nonVirtualCh.Key())).
							Entries(&resChannels).
							Exec(ctx, db),
					).To(Succeed())
					Expect(resChannels).To(HaveLen(2))

					Expect(resChannels[0].Name).To(Equal("NonVirtual"))
					Expect(resChannels[1].Name).To(Equal("UpdatedName"))
				},
			)
		})
	})

	Context("Name Validation Disabled", func() {
		Describe("Channel Creation", Ordered, func() {
			var (
				db  *gorp.DB
				svc *channel.Service
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				dist := testutil.NewNode(ctx)
				db = dist.DB
				svc = openService(
					ctx,
					dist,
					channel.ServiceConfig{ValidateNames: new(false)},
				)
			})
			It(
				"Should create a channel with spaces in the name",
				func(ctx SpecContext) {
					ch := channel.Channel{
						Name:        "my channel with spaces",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					Expect(ch.Key()).ToNot(BeZero())
					var retrieved channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entry(&retrieved).
						Exec(ctx, db)).To(Succeed())
					Expect(retrieved.Name).To(Equal("my channel with spaces"))
				},
			)
			It(
				"Should create a channel with special characters in the name",
				func(ctx SpecContext) {
					ch := channel.Channel{
						Name:        "sensor!@#$%",
						DataType:    telem.Float64T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					Expect(ch.Key()).ToNot(BeZero())
					var retrieved channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entry(&retrieved).
						Exec(ctx, db)).To(Succeed())
					Expect(retrieved.Name).To(Equal("sensor!@#$%"))
				},
			)
			It(
				"Should create a channel with a name starting with a digit",
				func(ctx SpecContext) {
					ch := channel.Channel{
						Name:        "1sensor",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					Expect(ch.Key()).ToNot(BeZero())
					var retrieved channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entry(&retrieved).
						Exec(ctx, db)).To(Succeed())
					Expect(retrieved.Name).To(Equal("1sensor"))
				},
			)
			It("Should still reject empty names", func(ctx SpecContext) {
				ch := channel.Channel{
					Name:        "",
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).
					To(MatchError(ContainSubstring("name: required")))
			})
			It("Should reject renaming to an empty name", func(ctx SpecContext) {
				ch := channel.Channel{
					Name:        "named",
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Rename(ctx, ch.Key(), "", false)
				})).To(MatchError(ContainSubstring("name: required")))
			})
			It(
				"Should allow renaming to a name with special characters",
				func(ctx SpecContext) {
					ch := channel.Channel{
						Name:        "original_name",
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Rename(
							ctx,
							ch.Key(),
							"new name with spaces!",
							false,
						)
					})).To(Succeed())
					var retrieved channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entry(&retrieved).
						Exec(ctx, db)).To(Succeed())
					Expect(retrieved.Name).To(Equal("new name with spaces!"))
				},
			)
		})
	})

	Describe("Delete", Ordered, func() {
		var (
			dist testutil.Node
			db   *gorp.DB
			svc  *channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist = testutil.NewNode(ctx)
			db = dist.DB
			svc = openService(ctx, dist)
		})
		Context("Single Channel", func() {
			var idxCh, ch channel.Channel
			JustBeforeEach(func(ctx SpecContext) {
				prefix := UniqueChannelName()
				idxCh.Name = prefix + "_time"
				idxCh.DataType = telem.TimestampT
				idxCh.IsIndex = true
				idxCh.Leaseholder = 1
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &idxCh)
				})).To(Succeed())
				ch.Name = prefix + "_data"
				ch.DataType = telem.Float64T
				ch.LocalIndex = idxCh.LocalKey
				ch.Leaseholder = 1
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
			})
			It(
				"Should not allow deletion of index channel with dependent channels",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, idxCh.Key(), true)
					})).To(
						MatchError(gorp.RestrictError{
							Target:    gorp.NewTarget("channel", idxCh.Key()),
							Holder:    gorp.NewTarget("channel", ch.Key()),
							Reference: "index",
						}),
					)
				},
			)
			It("Should delete the channel without error", func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).DeleteMany(
						ctx,
						channel.Keys{idxCh.Key(), ch.Key()},
						true,
					)
				})).To(Succeed())
			})
			It(
				"Should not be able to retrieve the channel after deletion",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, ch.Key(), true)
					})).To(Succeed())
					exists := MustSucceed(
						svc.NewRetrieve().
							Where(channel.MatchKeys(ch.Key())).
							Exists(ctx, db),
					)
					Expect(exists).To(BeFalse())
				},
			)
			It(
				"Should not be able to retrieve the channel from the time-series DB",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, ch.Key(), true)
					})).To(Succeed())
					Expect(
						dist.Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()),
					).
						Error().
						To(MatchError(ts.ErrChannelNotFound))
				},
			)
		})
		Context("Slow data removal", func() {
			It(
				"Should not block Gorp reads while removing data",
				func(ctx SpecContext) {
					base := DeferClose(storagetestutil.OpenLayer(ctx))
					fsys := xfstestutil.WrapFaultyFS(xfs.NewMem())
					tsDB := DeferClose(
						MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fsys))),
					)
					slow := testutil.NewNode(ctx, distribution.LayerConfig{
						Storage: &storage.Layer{KV: base.KV, TS: tsDB},
					})
					slowSvc := openService(ctx, slow)
					ch := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(slow.DB.WithTx(ctx, func(tx gorp.Tx) error {
						return slowSvc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					removing := make(chan struct{})
					release := make(chan struct{})
					DeferCleanup(sync.OnceFunc(func() { close(release) }))
					fsys.SetOptions(xfstestutil.WithBeforeRemove(sync.OnceFunc(func() {
						close(removing)
						<-release
					})))
					deleted := make(chan error, 1)
					go func() {
						deleted <- slow.DB.WithTx(ctx, func(tx gorp.Tx) error {
							return slowSvc.NewWriter(tx).Delete(ctx, ch.Key(), true)
						})
					}()
					Eventually(removing).Should(BeClosed())
					// The removal can start before the delete commits.
					Eventually(deleted).Should(Receive(BeNil()))
					read := make(chan bool, 1)
					go func() {
						defer GinkgoRecover()
						read <- MustSucceed(slowSvc.NewRetrieve().
							Where(channel.MatchKeys(ch.Key())).
							Exists(ctx, slow.DB))
					}()
					Eventually(read).Should(Receive(BeFalse()))
				},
			)
		})
		Context("Open channel", func() {
			It(
				"Should delete no channel when one is open for writing",
				func(ctx SpecContext) {
					chs := []channel.Channel{
						{
							Name:        UniqueChannelName(),
							DataType:    telem.TimestampT,
							IsIndex:     true,
							Leaseholder: 1,
						},
						{
							Name:        UniqueChannelName(),
							DataType:    telem.TimestampT,
							IsIndex:     true,
							Leaseholder: 1,
						},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).CreateMany(ctx, &chs)
					})).To(Succeed())
					kept, open := chs[0].Key(), chs[1].Key()
					Expect(dist.Storage.TS.WriteSeries(
						ctx,
						kept.StorageKey(),
						10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11, 12),
					)).To(Succeed())
					w := MustSucceed(dist.Storage.TS.OpenWriter(ctx, ts.WriterConfig{
						Start:    20 * telem.SecondTS,
						Channels: []ts.ChannelKey{open.StorageKey()},
					}))
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteMany(
							ctx,
							channel.Keys{kept, open},
							true,
						)
					})).To(MatchError(ContainSubstring("unclosed writers/iterators")))
					Expect(w.Close()).To(Succeed())
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(kept)).
						Exists(ctx, db)).To(BeTrue())
					fr := MustSucceed(
						dist.Storage.TS.Read(
							ctx,
							telem.TimeRangeMax,
							kept.StorageKey(),
						),
					)
					Expect(
						fr.Get(kept.StorageKey()).Series[0],
					).To(telem.MatchSeriesDataV(
						10*telem.SecondTS,
						11*telem.SecondTS,
						12*telem.SecondTS,
					))
				},
			)
		})
		Context("Calculated Channel Index", func() {
			It(
				"Should clear on open the index of a channel whose index does not exist",
				func(ctx SpecContext) {
					fresh := testutil.NewNode(ctx)
					calc := channel.Channel{
						Name:        "dangling_calc",
						Leaseholder: node.KeyFree,
						LocalKey:    7,
						LocalIndex:  8,
						DataType:    telem.Float64T,
						Expression:  "return 1 + 1",
						Virtual:     true,
					}
					Expect(gorp.WrapWriter[channel.Key, channel.Channel](fresh.DB).
						Set(ctx, calc)).To(Succeed())
					freshSvc := openService(ctx, fresh)
					var stored channel.Channel
					Expect(freshSvc.NewRetrieve().
						Where(channel.MatchKeys(calc.Key())).
						Entry(&stored).
						Exec(ctx, fresh.DB)).To(Succeed())
					Expect(stored.LocalIndex).To(BeZero())
				},
			)
		})
		Context("Nonexistent Channels", func() {
			It(
				"Should succeed when deleting a key that does not exist",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Delete(ctx, distchannel.NewKey(1, 40000), false)
					})).To(Succeed())
				},
			)
			It(
				"Should succeed when deleting multiple keys that do not exist",
				func(ctx SpecContext) {
					keys := channel.Keys{
						distchannel.NewKey(1, 40001),
						distchannel.NewKey(1, 40002),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteMany(ctx, keys, false)
					})).To(Succeed())
				},
			)
			It(
				"Should succeed when deleting by names that do not exist",
				func(ctx SpecContext) {
					names := []string{UniqueChannelName(), UniqueChannelName()}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteManyByNames(ctx, names, false)
					})).To(Succeed())
				},
			)
		})
		Context("Names", func() {
			It(
				"Should delete only exact names and not read a name as a pattern",
				func(ctx SpecContext) {
					prefix := UniqueChannelName()
					chs := []channel.Channel{
						{Name: prefix + "_a", DataType: telem.Float32T, Virtual: true},
						{Name: prefix + "_b", DataType: telem.Float32T, Virtual: true},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).CreateMany(ctx, &chs)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteManyByNames(
							ctx,
							[]string{prefix + "_a", prefix + "_.*"},
							false,
						)
					})).To(Succeed())
					var remaining []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(prefix+"_a", prefix+"_b")).
						Entries(&remaining).
						Exec(ctx, db)).To(Succeed())
					Expect(remaining).To(ConsistOf(HaveField("Name", prefix+"_b")))
				},
			)
		})
		Context("Internal Channels", func() {
			var internalCh channel.Channel
			JustBeforeEach(func(ctx SpecContext) {
				internalCh = channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Internal:    true,
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &internalCh)
				})).To(Succeed())
			})
			It(
				"Should return an error when deleting an internal channel by key",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, internalCh.Key(), false)
					})).
						To(MatchError(ContainSubstring("can't delete internal channel")))
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(internalCh.Key())).
						Exists(ctx, db)).To(BeTrue())
				},
			)
			It(
				"Should return an error when deleting an internal channel by name",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteManyByNames(
							ctx,
							[]string{internalCh.Name},
							false,
						)
					})).
						To(MatchError(ContainSubstring("can't delete internal channel")))
					Expect(svc.NewRetrieve().
						Where(channel.MatchNames(internalCh.Name)).
						Exists(ctx, db)).To(BeTrue())
				},
			)
			It(
				"Should not delete any channels when the batch contains an internal channel",
				func(ctx SpecContext) {
					externalCh := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.TimestampT,
						IsIndex:     true,
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &externalCh)
					})).To(Succeed())
					keys := channel.Keys{externalCh.Key(), internalCh.Key()}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).DeleteMany(ctx, keys, false)
					})).
						To(MatchError(ContainSubstring("can't delete internal channel")))
					var remaining []channel.Channel
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(keys...)).
						Entries(&remaining).
						Exec(ctx, db)).To(Succeed())
					Expect(remaining).To(HaveLen(2))
				},
			)
			It(
				"Should delete an internal channel when allowInternal is true",
				func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, internalCh.Key(), true)
					})).To(Succeed())
					Expect(svc.NewRetrieve().
						Where(channel.MatchKeys(internalCh.Key())).
						Exists(ctx, db)).To(BeFalse())
				},
			)
		})
	})

	Describe("Rename", Ordered, func() {
		var (
			dist testutil.Node
			db   *gorp.DB
			svc  *channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist = testutil.NewNode(ctx)
			db = dist.DB
			svc = openService(ctx, dist)
		})
		Context("Single channel", func() {
			var ch channel.Channel
			JustBeforeEach(func(ctx SpecContext) {
				ch.Virtual = true
				ch.Name = UniqueChannelName()
				ch.DataType = telem.Float64T
				ch.Leaseholder = 1
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
			})
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := UniqueChannelName()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Rename(ctx, ch.Key(), name, false)
				})).To(Succeed())
				var resCh channel.Channel
				Expect(svc.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, db)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
			Context("new name is invalid", func() {
				It("Should return an error", func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Rename(ctx, ch.Key(), "invalid name", false)
					})).To(MatchError(ContainSubstring("contains invalid characters")))
				})
			})
			Context("new name is empty", func() {
				It("Should return the same error as create", func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Rename(ctx, ch.Key(), "", false)
					})).To(MatchError(ContainSubstring("name: required")))
				})
			})
			Context("new name is a duplicate", func() {
				It("Should return an error", func(ctx SpecContext) {
					secondCh := channel.Channel{
						Name:     UniqueChannelName(),
						Virtual:  true,
						DataType: telem.Float64T,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &secondCh)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).
							Rename(ctx, ch.Key(), secondCh.Name, false)
					})).
						To(MatchError(ContainSubstring("channel with name '%s' already exists", secondCh.Name)))
				})
			})
		})
		Context("Multiple channels", func() {
			It("Should rename the channels without error", func(ctx SpecContext) {
				channels := []channel.Channel{
					{
						Name:     UniqueChannelName(),
						Virtual:  true,
						DataType: telem.Int64T,
					},
					{
						Name:     UniqueChannelName(),
						Virtual:  true,
						DataType: telem.Float32T,
					},
					{
						Name:        UniqueChannelName(),
						DataType:    telem.StringT,
						Leaseholder: node.KeyFree,
						Virtual:     true,
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).CreateMany(ctx, &channels)
				})).To(Succeed())
				keys := channel.KeysFromChannels(channels)
				names := []string{
					UniqueChannelName(),
					UniqueChannelName(),
					UniqueChannelName(),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).RenameMany(
						ctx,
						keys,
						names,
						false,
					)
				})).To(Succeed())
				var resChannels []channel.Channel
				Expect(
					svc.NewRetrieve().
						Where(channel.MatchKeys(keys...)).
						Entries(&resChannels).
						Exec(ctx, db),
				).To(Succeed())
				Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
				Expect(resChannels[0].Name).To(Equal(names[0]))
				Expect(resChannels[1].Name).To(Equal(names[1]))
				Expect(resChannels[2].Name).To(Equal(names[2]))
			})
			It(
				"Should send the renames of several writers to storage in one call",
				func(ctx SpecContext) {
					keys := MustSucceed(CreateVirtual(ctx, db, svc, 2))
					before := MustSucceed(
						dist.Storage.TS.RetrieveChannel(ctx, keys[0].StorageKey()),
					)
					first, second := UniqueChannelName(), UniqueChannelName()
					var between ts.Channel
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						if err := svc.NewWriter(tx).
							Rename(ctx, keys[0], first, false); err != nil {
							return err
						}
						// Storage takes no rename before the hook of the last one.
						gorp.BeforeCommit(tx, func(ctx context.Context) (err error) {
							between, err = dist.Storage.TS.RetrieveChannel(
								ctx,
								keys[0].StorageKey(),
							)
							return err
						})
						return svc.NewWriter(tx).Rename(ctx, keys[1], second, false)
					})).To(Succeed())
					Expect(between.Name).To(Equal(before.Name))
					Expect(MustSucceed(
						dist.Storage.TS.RetrieveChannel(ctx, keys[0].StorageKey()),
					).Name).To(Equal(first))
					Expect(MustSucceed(
						dist.Storage.TS.RetrieveChannel(ctx, keys[1].StorageKey()),
					).Name).To(Equal(second))
				},
			)
			It(
				"Should rename a channel created after an earlier rename",
				func(ctx SpecContext) {
					keys := MustSucceed(CreateVirtual(ctx, db, svc, 1))
					created := channel.Channel{
						Name:     UniqueChannelName(),
						Virtual:  true,
						DataType: telem.Int64T,
					}
					first, second := UniqueChannelName(), UniqueChannelName()
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						w := svc.NewWriter(tx)
						if err := w.Rename(ctx, keys[0], first, false); err != nil {
							return err
						}
						if err := w.Create(ctx, &created); err != nil {
							return err
						}
						return w.Rename(ctx, created.Key(), second, false)
					})).To(Succeed())
					Expect(MustSucceed(
						dist.Storage.TS.RetrieveChannel(ctx, keys[0].StorageKey()),
					).Name).To(Equal(first))
					Expect(MustSucceed(
						dist.Storage.TS.RetrieveChannel(
							ctx,
							created.Key().StorageKey(),
						),
					).Name).To(Equal(second))
				},
			)
		})
	})

	Describe("Limit", Ordered, func() {
		var (
			limit    = 5
			dist     testutil.Node
			db       *gorp.DB
			limitSvc *channel.Service
		)
		BeforeEach(func(ctx SpecContext) {
			dist = testutil.NewNode(ctx)
			db = dist.DB
			limitSvc = openService(
				ctx,
				dist,
				channel.ServiceConfig{Limit: fixedLimit(limit)},
			)
		})
		It("Should not allow creating channels over the limit", func(ctx SpecContext) {
			// Create channels up to the limit
			for i := range limit {
				ch := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        fmt.Sprintf("LimitTest%d", i),
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
			}

			// Try to create one more channel over the limit
			overLimitCh := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimestampT,
				Name:        "OverLimit",
				Leaseholder: 1,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return limitSvc.NewWriter(tx).Create(ctx, &overLimitCh)
			})).
				Error().To(MatchError(ContainSubstring("channel limit exceeded")))
		})

		It(
			"Should allow creating channels after deleting some to stay under the limit",
			func(ctx SpecContext) {
				// Create channels up to the limit
				channels := make([]channel.Channel, int(limit))
				for i := range limit {
					ch := channel.Channel{
						IsIndex:     true,
						DataType:    telem.TimestampT,
						Name:        fmt.Sprintf("LimitTest%d", i),
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					channels[i] = ch
				}

				// Try to create one more channel over the limit
				overLimitCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "OverLimit",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &overLimitCh)
				})).
					Error().To(MatchError(ContainSubstring("channel limit exceeded")))

				// Delete one channel
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Delete(ctx, channels[0].Key(), false)
				})).To(Succeed())

				// Now we should be able to create a new channel
				newCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "NewAfterDelete",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &newCh)
				})).To(Succeed())

				// Try to create one more channel (should fail again)
				anotherCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "AnotherOverLimit",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &anotherCh)
				})).
					Error().To(MatchError(ContainSubstring("channel limit exceeded")))
			},
		)

		It("Should allow retrieving channels even at the limit", func(ctx SpecContext) {
			// Create channels up to the limit
			createdChannels := make([]channel.Channel, int(limit))
			for i := range limit {
				ch := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        fmt.Sprintf("LimitTest%d", i),
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				createdChannels[i] = ch
			}

			// Try to create one more channel over the limit
			overLimitCh := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimestampT,
				Name:        "OverLimit",
				Leaseholder: 1,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return limitSvc.NewWriter(tx).Create(ctx, &overLimitCh)
			})).
				Error().To(MatchError(ContainSubstring("channel limit exceeded")))

			// Retrieve all channels - this should work fine even at the limit
			var retrievedChannels []channel.Channel
			retrieve := limitSvc.NewRetrieve()
			Expect(
				retrieve.Entries(&retrievedChannels).
					Where(channel.MatchLeaseholders(1)).
					Exec(ctx, db),
			).To(Succeed())
			Expect(retrievedChannels).To(HaveLen(limit))

			// Retrieve a specific channel by name
			var singleChannel channel.Channel
			Expect(
				retrieve.Where(channel.MatchKeys(createdChannels[0].Key())).
					Entry(&singleChannel).
					Exec(ctx, db),
			).To(Succeed())
			Expect(singleChannel.Name).To(Equal(createdChannels[0].Name))
		})
		It(
			"Should not allocate keys or storage for a create rejected by the limit",
			func(ctx SpecContext) {
				channels := make([]channel.Channel, int(limit))
				for i := range limit {
					ch := channel.Channel{
						IsIndex:     true,
						DataType:    telem.TimestampT,
						Name:        fmt.Sprintf("LimitTest%d", i),
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					channels[i] = ch
				}
				overLimitCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "OverLimit",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &overLimitCh)
				})).
					To(MatchError(ContainSubstring("channel limit exceeded")))

				// The rejected create must not leave an orphaned storage channel behind
				// at the key it would have been allocated.
				nextKey := distchannel.NewKey(1, channels[limit-1].LocalKey+1)
				Expect(dist.Storage.TS.RetrieveChannels(ctx, nextKey.StorageKey())).
					Error().To(MatchError(ts.ErrChannelNotFound))

				// The rejected create must not burn a local key: after freeing
				// capacity, the next create receives the next sequential key.
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Delete(ctx, channels[0].Key(), false)
				})).To(Succeed())
				newCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "NewAfterReject",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &newCh)
				})).To(Succeed())
				Expect(newCh.LocalKey).To(Equal(channels[limit-1].LocalKey + 1))
			},
		)
		It(
			"Should not edit the channel limit if a deletion fails in TS",
			func(ctx SpecContext) {
				createdChannels := make([]channel.Channel, int(limit))
				for i := range limit {
					ch := channel.Channel{
						IsIndex:     true,
						DataType:    telem.TimestampT,
						Name:        fmt.Sprintf("LimitTest%d", i),
						Leaseholder: 1,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return limitSvc.NewWriter(tx).Create(ctx, &ch)
					})).To(Succeed())
					createdChannels[i] = ch
				}
				MustOpen(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Keys: []channel.Key{createdChannels[0].Key()},
				}))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).
						Delete(ctx, createdChannels[0].Key(), false)
				})).
					To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
				newCh := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Name:        "NewAfterDelete",
					Leaseholder: 1,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return limitSvc.NewWriter(tx).Create(ctx, &newCh)
				})).
					To(MatchError(ContainSubstring("channel limit exceeded")))
			},
		)
	})

	Describe("Storage", Ordered, func() {
		var (
			dist     testutil.Node
			db       *gorp.DB
			svc      *channel.Service
			rollback = errors.New("rollback")
			create   = func(ctx SpecContext) channel.Channel {
				GinkgoHelper()
				ch := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				return ch
			}
			stored = func(ctx SpecContext, ch channel.Channel) ts.Channel {
				GinkgoHelper()
				return MustSucceed(
					dist.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
				)
			}
			expectNotStored = func(ctx SpecContext, ch channel.Channel) {
				GinkgoHelper()
				Expect(dist.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey())).
					Error().To(MatchError(ts.ErrChannelNotFound))
			}
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist = testutil.NewNode(ctx)
			db = dist.DB
			svc = openService(ctx, dist)
		})
		It(
			"Should create the storage channel when the transaction commits",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).Create(ctx, &ch); err != nil {
						return err
					}
					expectNotStored(ctx, ch)
					return nil
				})).To(Succeed())
				Expect(stored(ctx, ch).Name).To(Equal(ch.Name))
			},
		)
		It(
			"Should create no storage channel when the transaction rolls back",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).Create(ctx, &ch); err != nil {
						return err
					}
					return rollback
				})).To(MatchError(rollback))
				Expect(ch.LocalKey).ToNot(BeZero())
				expectNotStored(ctx, ch)
			},
		)
		It(
			"Should delete the storage channel when the transaction commits",
			func(ctx SpecContext) {
				ch := create(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).
						Delete(ctx, ch.Key(), false); err != nil {
						return err
					}
					stored(ctx, ch)
					return nil
				})).To(Succeed())
				expectNotStored(ctx, ch)
			},
		)
		It(
			"Should keep the storage channel when a delete rolls back",
			func(ctx SpecContext) {
				ch := create(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).
						Delete(ctx, ch.Key(), false); err != nil {
						return err
					}
					return rollback
				})).To(MatchError(rollback))
				stored(ctx, ch)
			},
		)
		It(
			"Should keep the channel when its storage channel cannot be deleted",
			func(ctx SpecContext) {
				ch := create(ctx)
				MustOpen(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Keys: []channel.Key{ch.Key()},
				}))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
				})).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
				Expect(svc.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Exists(ctx, db)).To(BeTrue())
			},
		)
		It(
			"Should rename the storage channel when the transaction commits",
			func(ctx SpecContext) {
				ch := create(ctx)
				name := UniqueChannelName()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).
						Rename(ctx, ch.Key(), name, false); err != nil {
						return err
					}
					Expect(stored(ctx, ch).Name).To(Equal(ch.Name))
					return nil
				})).To(Succeed())
				Expect(stored(ctx, ch).Name).To(Equal(name))
				var res channel.Channel
				Expect(svc.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal(name))
			},
		)
	})

	Describe("ChangeDataType", Ordered, func() {
		var (
			db  *gorp.DB
			svc *channel.Service
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			dist := testutil.NewNode(ctx)
			db = dist.DB
			svc = openService(ctx, dist)
		})
		It(
			"Should change the data type of a calculated channel",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return 1",
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &base)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						ChangeDataType(ctx, base.Key(), telem.Float32T)
				})).To(Succeed())
				var retrieved channel.Channel
				Expect(
					svc.NewRetrieve().
						Where(channel.MatchKeys(base.Key())).
						Entry(&retrieved).
						Exec(ctx, db),
				).To(Succeed())
				Expect(retrieved.DataType).To(Equal(telem.Float32T))
			},
		)
		It(
			"Should return a validation error when changing the data type of a non-calculated channel",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						ChangeDataType(ctx, ch.Key(), telem.Float32T)
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring(
							"cannot change the data type of non-calculated channel",
						),
					),
				))
				var retrieved channel.Channel
				Expect(
					svc.NewRetrieve().
						Where(channel.MatchKeys(ch.Key())).
						Entry(&retrieved).
						Exec(ctx, db),
				).To(Succeed())
				Expect(retrieved.DataType).To(Equal(telem.Float64T))
			},
		)
	})
})
