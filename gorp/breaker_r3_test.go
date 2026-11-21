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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var errB3 = errors.New("breaker r3: commit refused")

type b3Item struct {
	Key    int32
	Name   string
	Labels []string
}

func (i b3Item) GorpKey() int32 { return i.Key }

type b3Label struct{ Key string }

func (l b3Label) GorpKey() string { return l.Key }

var _ = Describe("Breaker r3 ended: a transaction past a failed commit", func() {
	var (
		db     *gorp.DB
		items  *gorp.Table[int32, b3Item]
		labels *gorp.Table[string, b3Label]
		lookup *gorp.LookupIndex[int32, b3Item, string]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		labels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b3Label]{
			DB:   db,
			Name: "b3label",
		}))
		lookup = gorp.NewLookupIndex[int32, b3Item, string](
			"name",
			func(e *b3Item) string { return e.Name },
		)
		ref := gorp.NewKeyReference(gorp.KeyReferenceConfig[int32, b3Item, string]{
			Name:     "labels",
			Table:    "b3label",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets:  func(e *b3Item) []string { return e.Labels },
		})
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, b3Item]{
			DB:         db,
			Name:       "b3item",
			Indexes:    []gorp.Index[int32, b3Item]{lookup},
			References: []gorp.Reference[int32, b3Item]{ref},
		}))
	})

	failCommit := func(ctx context.Context, tx gorp.Tx) {
		gorp.BeforeCommit(tx, func(context.Context) error { return errB3 })
		Expect(tx.Commit(ctx)).To(MatchError(errB3))
	}

	It(
		"Should find an entry written past a failed commit through an index filter of the same transaction",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			DeferCleanup(tx.Close)
			failCommit(ctx, tx)
			Expect(items.NewCreate().Entry(&b3Item{Key: 2, Name: "b"}).Exec(ctx, tx)).
				To(Succeed())
			var byKey []b3Item
			Expect(items.NewRetrieve().
				Where(gorp.MatchKeys[int32, b3Item](2)).
				Entries(&byKey).
				Exec(ctx, tx)).To(Succeed())
			Expect(byKey).To(HaveLen(1))
			var byName []b3Item
			Expect(items.NewRetrieve().
				Where(lookup.Filter("b")).
				Entries(&byName).
				Exec(ctx, tx)).To(Succeed())
			Expect(byName).To(Equal(byKey))
		},
	)

	It(
		"Should refuse to delete a label that an entry written past a failed commit holds",
		func(ctx SpecContext) {
			Expect(labels.NewCreate().Entry(&b3Label{Key: "l"}).Exec(ctx, db)).
				To(Succeed())
			tx := db.OpenTx()
			DeferCleanup(tx.Close)
			failCommit(ctx, tx)
			Expect(items.NewCreate().
				Entry(&b3Item{Key: 1, Name: "a", Labels: []string{"l"}}).
				Exec(ctx, tx)).To(Succeed())
			Expect(labels.NewDelete().
				Where(gorp.MatchKeys[string, b3Label]("l")).
				Exec(ctx, tx)).To(MatchError(ContainSubstring(
				"cannot delete b3label l because b3item 1 references it",
			)))
		},
	)

	It(
		"Should list the links of an entry written past a failed commit in the changes of the transaction",
		func(ctx SpecContext) {
			Expect(labels.NewCreate().Entry(&b3Label{Key: "l"}).Exec(ctx, db)).
				To(Succeed())
			tx := db.OpenTx()
			DeferCleanup(tx.Close)
			failCommit(ctx, tx)
			Expect(items.NewCreate().
				Entry(&b3Item{Key: 1, Name: "a", Labels: []string{"l"}}).
				Exec(ctx, tx)).To(Succeed())
			var item gorp.Change
			for _, c := range gorp.ChangesOf(tx) {
				if c.Record.Table == "b3item" {
					item = c
				}
			}
			Expect(item.Links).To(HaveLen(1))
			Expect(
				item.Links[0].Target,
			).To(Equal(gorp.Target{Table: "b3label", Key: "l"}))
		},
	)
})

var _ = Describe("Breaker r3 race: a read of a transaction while it closes", func() {
	It(
		"Should let another goroutine read the index of a transaction while it closes",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			lookup := gorp.NewLookupIndex[int32, b3Item, string](
				"name",
				func(e *b3Item) string { return e.Name },
			)
			items := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, b3Item]{
				DB:      db,
				Name:    "b3item",
				Indexes: []gorp.Index[int32, b3Item]{lookup},
			}))
			entries := make([]b3Item, 256)
			for i := range entries {
				entries[i] = b3Item{Key: int32(i), Name: fmt.Sprintf("n%d", i)}
			}
			tx := db.OpenTx()
			Expect(items.NewCreate().Entries(&entries).Exec(ctx, tx)).To(Succeed())
			read, done := make(chan struct{}), make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				for i := range 64 {
					_, err := lookup.Get(ctx, tx, "n1")
					Expect(err).ToNot(HaveOccurred())
					if i == 0 {
						close(read)
					}
				}
			}()
			<-read
			Expect(tx.Close()).To(Succeed())
			<-done
		},
	)
})
