// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"slices"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

type weakChannel struct{ Key uint32 }

func (c weakChannel) GorpKey() uint32 { return c.Key }

type weakSymbol struct {
	Key      string
	Channels []uint32
}

func (s weakSymbol) GorpKey() string { return s.Key }

type weakNote struct {
	Key    string
	Target *gorp.Target
}

func (n weakNote) GorpKey() string { return n.Key }

const weakTag uint8 = 7

func newWeakChannelsRef() *gorp.KeyReference[string, weakSymbol, uint32] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[string, weakSymbol, uint32]{
		Name:     "channels",
		Table:    "channel",
		OnDelete: gorp.Weak,
		Kind:     gorp.Use,
		Tag:      weakTag,
		Targets:  func(s *weakSymbol) []uint32 { return s.Channels },
		Detach: func(s *weakSymbol, k uint32) {
			s.Channels = slices.DeleteFunc(
				slices.Clone(s.Channels),
				func(v uint32) bool { return v == k },
			)
		},
	})
}

var _ = Describe("Weak reference", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, weakChannel]
		symbols  *gorp.Table[string, weakSymbol]
		// counter is the highest key the channel table issued.
		counter atomic.Uint32
		issued  = func(k uint32) bool { return k <= counter.Load() }
		open    = func(ctx SpecContext) {
			GinkgoHelper()
			db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			channels = MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[uint32, weakChannel]{
					DB:     db,
					Name:   "channel",
					Issued: issued,
				}),
			)
			symbols = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, weakSymbol]{
				DB:   db,
				Name: "symbol",
				References: []gorp.Reference[string, weakSymbol]{
					newWeakChannelsRef(),
				},
			}))
		}
		createChannel = func(ctx SpecContext, k uint32) {
			GinkgoHelper()
			counter.Store(max(counter.Load(), k))
			Expect(channels.NewCreate().Entry(&weakChannel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
		deleteChannel = func(ctx SpecContext, k uint32) {
			GinkgoHelper()
			Expect(channels.NewDelete().
				Where(gorp.MatchKeys[uint32, weakChannel](k)).
				Exec(ctx, db)).To(Succeed())
		}
		retrieveSymbol = func(ctx SpecContext, k string) weakSymbol {
			GinkgoHelper()
			var s weakSymbol
			Expect(symbols.NewRetrieve().
				Where(gorp.MatchKeys[string, weakSymbol](k)).
				Entry(&s).
				Exec(ctx, db)).To(Succeed())
			return s
		}
	)

	BeforeEach(func() { counter.Store(0) })

	Describe("Delete", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should keep the holder and the deleted key", func(ctx SpecContext) {
			createChannel(ctx, 1)
			createChannel(ctx, 2)
			Expect(symbols.NewCreate().
				Entry(&weakSymbol{Key: "a", Channels: []uint32{1, 2}}).
				Exec(ctx, db)).To(Succeed())
			deleteChannel(ctx, 1)
			Expect(retrieveSymbol(ctx, "a").Channels).To(Equal([]uint32{1, 2}))
		})

		It("Should keep the holder in the dependents of the deleted key", func(
			ctx SpecContext,
		) {
			createChannel(ctx, 1)
			Expect(symbols.NewCreate().
				Entry(&weakSymbol{Key: "a", Channels: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			deleteChannel(ctx, 1)
			Expect(channels.Dependents(ctx, db, 1)).To(Equal([]gorp.Dependent{{
				Holder:    gorp.Target{Table: "symbol", Key: "a"},
				Reference: "channels",
				Kind:      gorp.Use,
				Tag:       weakTag,
			}}))
		})
	})

	Describe("Write", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should accept a key that was issued and deleted", func(ctx SpecContext) {
			createChannel(ctx, 1)
			deleteChannel(ctx, 1)
			Expect(symbols.NewCreate().
				Entry(&weakSymbol{Key: "a", Channels: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			Expect(retrieveSymbol(ctx, "a").Channels).To(Equal([]uint32{1}))
		})

		It("Should reject a key that was never issued", func(ctx SpecContext) {
			createChannel(ctx, 1)
			Expect(symbols.NewCreate().
				Entry(&weakSymbol{Key: "a", Channels: []uint32{2}}).
				Exec(ctx, db)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("channels: channel 2 was never issued: validation error"),
			))
		})

		It("Should check a target reference against Issued", func(ctx SpecContext) {
			notes := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, weakNote]{
				DB:   db,
				Name: "note",
				References: []gorp.Reference[string, weakNote]{
					gorp.NewTargetReference(
						gorp.TargetReferenceConfig[string, weakNote]{
							Name:     "target",
							Tables:   []string{"channel"},
							OnDelete: gorp.Weak,
							Kind:     gorp.Use,
							Targets: func(n *weakNote) []gorp.Target {
								if n.Target == nil {
									return nil
								}
								return []gorp.Target{*n.Target}
							},
						},
					),
				},
			}))
			createChannel(ctx, 1)
			deleteChannel(ctx, 1)
			Expect(notes.NewCreate().
				Entry(&weakNote{Key: "a", Target: &gorp.Target{Table: "channel", Key: "1"}}).
				Exec(ctx, db)).To(Succeed())
			Expect(notes.NewCreate().
				Entry(&weakNote{Key: "b", Target: &gorp.Target{Table: "channel", Key: "2"}}).
				Exec(ctx, db),
			).To(MatchError(ContainSubstring("channel 2 was never issued")))
		})
	})

	Describe("DropDangling", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should keep a deleted key", func(ctx SpecContext) {
			createChannel(ctx, 1)
			deleteChannel(ctx, 1)
			s := weakSymbol{Key: "a", Channels: []uint32{1}}
			Expect(symbols.DropDangling(ctx, db, &s)).To(BeFalse())
			Expect(s.Channels).To(Equal([]uint32{1}))
		})
	})

	Describe("Registration", func() {
		BeforeEach(func() {
			db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
		})

		It("Should reject a holder whose open target table has no Issued", func(
			ctx SpecContext,
		) {
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, weakChannel]{
				DB:   db,
				Name: "channel",
			}))
			Expect(gorp.OpenTable(ctx, gorp.TableConfig[string, weakSymbol]{
				DB:   db,
				Name: "symbol",
				References: []gorp.Reference[string, weakSymbol]{
					newWeakChannelsRef(),
				},
			})).Error().To(MatchError(
				`[gorp] - weak reference channels of table "symbol" points at ` +
					`table "channel", which has no Issued`,
			))
		})

		It(
			"Should reject a target table with no Issued that opens after the holder",
			func(
				ctx SpecContext,
			) {
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, weakSymbol]{
					DB:   db,
					Name: "symbol",
					References: []gorp.Reference[string, weakSymbol]{
						newWeakChannelsRef(),
					},
				}))
				Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, weakChannel]{
					DB:   db,
					Name: "channel",
				})).Error().To(MatchError(
					`[gorp] - weak reference channels of table "symbol" points at ` +
						`table "channel", which has no Issued`,
				))
			},
		)

		It(
			"Should free the holder's name after a rejected open",
			func(ctx SpecContext) {
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, weakChannel]{
					DB:   db,
					Name: "channel",
				}))
				Expect(gorp.OpenTable(ctx, gorp.TableConfig[string, weakSymbol]{
					DB:   db,
					Name: "symbol",
					References: []gorp.Reference[string, weakSymbol]{
						newWeakChannelsRef(),
					},
				})).Error().To(MatchError(ContainSubstring("which has no Issued")))
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, weakSymbol]{
					DB:   db,
					Name: "symbol",
				}))
			},
		)
	})
})
