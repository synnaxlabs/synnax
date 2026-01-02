// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("KV", Ordered, func() {
	var (
		db        *gorp.DB
		rangerSvc *ranger.Service
		kvSvc     *kv.Service
		ctx       context.Context
		otg       *ontology.Ontology
		tx        gorp.Tx
		closer    io.Closer
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		ctx = context.Background()
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB:           db,
			EnableSearch: config.True(),
		}))
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
		lab := MustSucceed(label.OpenService(ctx, label.ServiceConfig{DB: db, Ontology: otg, Group: g}))
		rangerSvc = MustSucceed(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lab,
		}))
		kvSvc = MustSucceed(kv.OpenService(ctx, kv.ServiceConfig{
			DB: db,
		}))
		closer = xio.MultiCloser{db, otg, g, rangerSvc, kvSvc}
	})
	AfterAll(func() {
		Expect(closer.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	It("Should be able to store key-value pairs in a range", func() {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Set(ctx, r.Key, "key", "value")).To(Succeed())
	})

	It("Should be able to retrieve key-value pairs from a range", func() {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Set(ctx, r.Key, "key", "value")).To(Succeed())
		value, err := kvSvc.NewReader(tx).Get(ctx, r.Key, "key")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("value"))
	})

	It("Should be able to delete key-value pairs from a range", func() {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Set(ctx, r.Key, "key", "value")).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Delete(ctx, r.Key, "key")).To(Succeed())
		_, err := kvSvc.NewReader(tx).Get(ctx, r.Key, "key")
		Expect(err).To(HaveOccurred())
	})

	It("Should set many key-value pairs on the range", func() {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).SetMany(ctx, r.Key, []kv.Pair{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
		})).To(Succeed())
		value, err := kvSvc.NewReader(tx).Get(ctx, r.Key, "key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("value1"))
		value, err = kvSvc.NewReader(tx).Get(ctx, r.Key, "key2")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("value2"))
	})

	It("Should be able to list all key-value pairs in a range", func() {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Set(ctx, r.Key, "key1", "value1")).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Set(ctx, r.Key, "key2", "value2")).To(Succeed())
		pairs, err := kvSvc.NewReader(tx).List(ctx, r.Key)
		Expect(err).ToNot(HaveOccurred())
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
		Expect(kvSvc.NewWriter(tx).Delete(ctx, r.Key, "key1")).To(Succeed())
		pairs, err = kvSvc.NewReader(tx).List(ctx, r.Key)
		Expect(err).ToNot(HaveOccurred())
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
	})
})
