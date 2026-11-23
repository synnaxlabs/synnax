// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Interner", func() {
	var (
		svc *enumeration.Service
		tx  gorp.Tx
		in  *enumeration.Interner
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
		in = svc.NewWriter(tx).NewInterner()
	})
	red := &color.Ref{Variant: color.FixedRef{Value: xcolor.Color{R: 255, A: 255}}}
	retrieve := func(ctx SpecContext, key enumeration.Key) enumeration.Enumeration {
		GinkgoHelper()
		var e enumeration.Enumeration
		Expect(svc.NewRetrieve().
			Where(enumeration.MatchKeys(key)).
			Entry(&e).
			Exec(ctx, tx)).To(Succeed())
		return e
	}
	// added returns how many enumerations the spec added beside the built-ins.
	added := func(ctx SpecContext) int {
		GinkgoHelper()
		var all []enumeration.Enumeration
		Expect(svc.NewRetrieve().Entries(&all).Exec(ctx, tx)).To(Succeed())
		return len(all) - len(enumeration.Builtins())
	}

	It("Should create an enumeration with the name and entries", func(ctx SpecContext) {
		entries := []enumeration.Entry{
			{Key: "off", Value: 0, Name: "Off"},
			{Key: "on", Value: 1, Name: "On", Color: red},
		}
		key := MustSucceed(in.Intern(ctx, "Valve", entries))
		e := retrieve(ctx, key)
		Expect(e.Name).To(Equal("Valve"))
		Expect(e.Entries).To(Equal(entries))
	})

	It("Should share one enumeration between lists in another order with other keys",
		func(ctx SpecContext) {
			first := MustSucceed(in.Intern(ctx, "Valve", []enumeration.Entry{
				{Key: "off", Value: 0, Name: "Off"},
				{Key: "on", Value: 1, Name: "On", Color: red},
			}))
			second := MustSucceed(in.Intern(ctx, "Pump", []enumeration.Entry{
				{Key: "b", Value: 1, Name: "On", Color: red},
				{Key: "a", Value: 0, Name: "Off"},
			}))
			Expect(second).To(Equal(first))
			Expect(added(ctx)).To(Equal(1))
		},
	)

	DescribeTable(
		"Should keep lists apart when they differ",
		func(ctx SpecContext, other enumeration.Entry) {
			first := MustSucceed(in.Intern(ctx, "Valve", []enumeration.Entry{
				{Key: "off", Value: 0, Name: "Off"},
			}))
			second := MustSucceed(in.Intern(ctx, "Valve", []enumeration.Entry{other}))
			Expect(second).ToNot(Equal(first))
			Expect(added(ctx)).To(Equal(2))
		},
		Entry("in value", enumeration.Entry{Key: "off", Value: 1, Name: "Off"}),
		Entry("in name", enumeration.Entry{Key: "off", Value: 0, Name: "Closed"}),
		Entry(
			"in color",
			enumeration.Entry{Key: "off", Value: 0, Name: "Off", Color: red},
		),
	)

	It("Should keep the first entry of a repeated value", func(ctx SpecContext) {
		key := MustSucceed(in.Intern(ctx, "Valve", []enumeration.Entry{
			{Key: "off", Value: 0, Name: "Off"},
			{Key: "closed", Value: 0, Name: "Closed"},
		}))
		Expect(retrieve(ctx, key).Entries).
			To(Equal([]enumeration.Entry{{Key: "off", Value: 0, Name: "Off"}}))
	})
})
