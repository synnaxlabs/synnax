// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("KV", Ordered, func() {
	var (
		db        *gorp.DB
		rangerSvc *ranger.Service
		kvSvc     *kv.Service
		otg       *ontology.Ontology
		tx        gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		lab := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lab,
		}))
		kvSvc = MustOpen(kv.OpenService(ctx, kv.ServiceConfig{
			DB: db,
		}))
	})
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	It("Should be able to store key-value pairs in a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
	})

	It(
		"Should be able to retrieve key-value pairs from a range",
		func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(
				kvSvc.NewWriter(tx).
					SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
			).To(Succeed())
			Expect(kvSvc.NewReader(tx).GetMany(ctx, r.Key, []string{"key"})).
				To(Equal([]kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}))
		},
	)

	It("Should be able to delete key-value pairs from a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Delete(ctx, r.Key, "key")).To(Succeed())
		Expect(kvSvc.NewReader(tx).List(ctx, r.Key)).To(BeEmpty())
	})

	It("Should set many key-value pairs on the range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).SetMany(ctx, []kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		})).To(Succeed())
		Expect(
			kvSvc.NewReader(tx).GetMany(ctx, r.Key, []string{"key1", "key2"}),
		).To(ConsistOf(
			kv.Pair{Range: r.Key, Key: "key1", Value: "value1"},
			kv.Pair{Range: r.Key, Key: "key2", Value: "value2"},
		))
	})

	It("Should be able to list all key-value pairs in a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		kvW := kvSvc.NewWriter(tx)
		Expect(kvW.SetMany(ctx, []kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		})).To(Succeed())
		pairs := MustSucceed(kvSvc.NewReader(tx).List(ctx, r.Key))
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
		Expect(kvW.Delete(ctx, r.Key, "key1")).To(Succeed())
		pairs = MustSucceed(kvSvc.NewReader(tx).List(ctx, r.Key))
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
	})
})
