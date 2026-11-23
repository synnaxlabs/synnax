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
	"context"
	"math/rand/v2"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Held breaker", func() {
	var (
		db       *gorp.DB
		devices  *gorp.Table[string, refDevice]
		refs     deviceRefs
		rackIDs  = []uint32{1, 2, 3, 4, 5}
		labelIDs = []string{"l1", "l2", "l3", "l4"}
	)
	openOn := func(ctx SpecContext, kvDB *gorp.DB) {
		GinkgoHelper()
		db = kvDB
		StartRun(db.Run)
		racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
		}))
		labels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		refs = newDeviceRefs()
		devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
			DB:         db,
			Name:       "device",
			References: refs.all(),
		}))
		for _, r := range rackIDs {
			Expect(racks.NewCreate().Entry(&refRack{Key: r}).Exec(ctx, db)).
				To(Succeed())
		}
		for _, l := range labelIDs {
			Expect(labels.NewCreate().Entry(&refLabel{Key: l}).Exec(ctx, db)).
				To(Succeed())
		}
	}
	// expected reads every device through snap and returns the racks and labels that
	// at least one device holds.
	expected := func(ctx SpecContext, snap gorp.Snapshot) ([]uint32, []string) {
		GinkgoHelper()
		var all []refDevice
		Expect(devices.NewRetrieve().Entries(&all).Exec(ctx, snap)).To(Succeed())
		racks, labels := make(set.Set[uint32]), make(set.Set[string])
		for _, d := range all {
			racks.Add(d.Rack)
			labels.Add(d.Labels...)
		}
		return racks.Slice(), labels.Slice()
	}
	check := func(ctx SpecContext, snap gorp.Snapshot, msg string) {
		GinkgoHelper()
		racks, labels := expected(ctx, snap)
		Expect(refs.rack.Held(ctx, snap)).To(ConsistOf(racks), "rack: "+msg)
		Expect(refs.labels.Held(ctx, snap)).To(ConsistOf(labels), "labels: "+msg)
	}
	randomDevice := func(r *rand.Rand, pool int) refDevice {
		d := refDevice{
			Key:  "d" + strconv.Itoa(r.IntN(pool)),
			Rack: rackIDs[r.IntN(len(rackIDs))],
		}
		for _, l := range labelIDs {
			if r.IntN(3) == 0 {
				d.Labels = append(d.Labels, l)
			}
		}
		return d
	}

	Describe("Index path", func() {
		BeforeEach(func(ctx SpecContext) {
			openOn(ctx, DeferClose(wrapMsgpack(memkv.New())))
		})

		It("Should return nothing for an empty table", func(ctx SpecContext) {
			Expect(refs.rack.Held(ctx, db)).To(BeEmpty())
			tx := DeferClose(db.OpenTx())
			Expect(refs.rack.Held(ctx, tx)).To(BeEmpty())
			Expect(refs.labels.Held(ctx, tx)).To(BeEmpty())
		})

		It("Should match a scan after random staged writes", func(ctx SpecContext) {
			for seed := range uint64(48) {
				r := rand.New(rand.NewPCG(seed, 11))
				pool := 6 + r.IntN(50)
				for round := range 6 {
					tx := db.OpenTx()
					for range 1 + r.IntN(20) {
						if r.IntN(3) == 0 {
							k := "d" + strconv.Itoa(r.IntN(pool))
							Expect(devices.NewDelete().
								Where(gorp.MatchKeys[string, refDevice](k)).
								Exec(ctx, tx)).To(Succeed())
						} else {
							d := randomDevice(r, pool)
							Expect(devices.NewCreate().Entry(&d).Exec(ctx, tx)).
								To(Succeed())
						}
						check(ctx, tx, "seed "+strconv.Itoa(int(seed))+
							" round "+strconv.Itoa(round))
					}
					if r.IntN(2) == 0 {
						Expect(tx.Commit(ctx)).To(Succeed())
					}
					Expect(tx.Close()).To(Succeed())
					check(ctx, db, "committed seed "+strconv.Itoa(int(seed)))
				}
			}
		})

		It("Should hold a target with many holders until the last one goes", func(
			ctx SpecContext,
		) {
			for i := range 40 {
				Expect(devices.NewCreate().
					Entry(&refDevice{Key: "m" + strconv.Itoa(i), Rack: 1}).
					Exec(ctx, db)).To(Succeed())
			}
			tx := DeferClose(db.OpenTx())
			for i := range 39 {
				Expect(devices.NewCreate().
					Entry(&refDevice{Key: "m" + strconv.Itoa(i), Rack: 2}).
					Exec(ctx, tx)).To(Succeed())
			}
			check(ctx, tx, "one holder left")
			Expect(devices.NewDelete().
				Where(gorp.MatchKeys[string, refDevice]("m39")).
				Exec(ctx, tx)).To(Succeed())
			check(ctx, tx, "no holder left")
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "m39", Rack: 1}).
				Exec(ctx, tx)).To(Succeed())
			check(ctx, tx, "holder back")
		})

		It("Should end a wait on the index when its context ends", func(
			ctx SpecContext,
		) {
			kvs := &blockingDB{DB: memkv.New(), release: make(chan struct{})}
			blocked := DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
			StartRun(blocked.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   blocked,
				Name: "rack",
			}))
			blockedRefs := newDeviceRefs()
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         blocked,
				Name:       "device",
				References: blockedRefs.all(),
			}))
			DeferCleanup(func() { close(kvs.release) })
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			Expect(blockedRefs.rack.Held(canceled, nil)).Error().
				To(MatchError(context.Canceled))
		})
	})

	Describe("Commits and loads", func() {
		It("Should drop the target of a commit that fails", func(ctx SpecContext) {
			store := &failingCommitDB{DB: memkv.New()}
			openOn(ctx, DeferClose(wrapMsgpack(store)))
			var during []uint32
			store.during = func() { during = MustSucceed(refs.rack.Held(ctx, nil)) }
			store.fail = errors.New("commit boom")
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "a", Rack: 4}).
				Exec(ctx, db)).To(MatchError(ContainSubstring("commit boom")))
			Expect(during).To(ConsistOf(uint32(4)))
			Expect(refs.rack.Held(ctx, nil)).To(BeEmpty())
			store.during, store.fail = nil, nil
			check(ctx, db, "after the failed commit")
		})

		It("Should answer a read that waits for the index to load", func(
			ctx SpecContext,
		) {
			kvs := &blockingDB{DB: memkv.New(), release: make(chan struct{})}
			pre := gorp.Wrap(kvs.DB, gorp.WithCodec(msgpack.Codec))
			Expect(gorp.NewCreate[uint32, refRack]().
				Entries(&[]refRack{{Key: 1}, {Key: 2}}).
				Exec(ctx, pre)).To(Succeed())
			Expect(gorp.NewCreate[string, refDevice]().
				Entries(&[]refDevice{{Key: "a", Rack: 2}, {Key: "b", Rack: 2}}).
				Exec(ctx, pre)).To(Succeed())
			loading := DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
			StartRun(loading.Run)
			loadingRefs := newDeviceRefs()
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         loading,
				Name:       "device",
				References: loadingRefs.all(),
			}))
			done := make(chan []uint32)
			go func() {
				defer GinkgoRecover()
				done <- MustSucceed(loadingRefs.rack.Held(ctx, loading))
			}()
			Consistently(done).ShouldNot(Receive())
			close(kvs.release)
			Eventually(done).Should(Receive(ConsistOf(uint32(2))))
		})
	})

	Describe("Scan path", func() {
		It("Should give the same answer as the index after random writes", func(
			ctx SpecContext,
		) {
			kvs := DeferClose(memkv.New())
			pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
			for _, r := range rackIDs {
				Expect(gorp.NewCreate[uint32, refRack]().
					Entry(&refRack{Key: r}).
					Exec(ctx, pre)).To(Succeed())
			}
			Expect(gorp.NewCreate[string, refDevice]().
				Entry(&refDevice{Key: "d0", Rack: 1}).
				Exec(ctx, pre)).To(Succeed())
			failing := gorp.Wrap(
				&failOnceDB{DB: kvs, err: errors.New("populate boom")},
				gorp.WithCodec(msgpack.Codec),
			)
			db = failing
			StartRun(db.Run)
			refs = newDeviceRefs()
			devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         db,
				Name:       "device",
				References: refs.all(),
			}))
			Expect(refs.rack.Get(ctx, nil, 1)).Error().
				To(MatchError(gorp.ErrIndexInvalid))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			r := rand.New(rand.NewPCG(3, 5))
			for range 10 {
				tx := db.OpenTx()
				for range 1 + r.IntN(10) {
					d := refDevice{
						Key:  "d" + strconv.Itoa(r.IntN(8)),
						Rack: rackIDs[r.IntN(len(rackIDs))],
					}
					Expect(devices.NewCreate().Entry(&d).Exec(ctx, tx)).
						To(Succeed())
					racks, _ := expected(ctx, tx)
					Expect(refs.rack.Held(ctx, tx)).To(ConsistOf(racks))
				}
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				racks, _ := expected(ctx, db)
				Expect(refs.rack.Held(ctx, db)).To(ConsistOf(racks))
			}
		})
	})
})
