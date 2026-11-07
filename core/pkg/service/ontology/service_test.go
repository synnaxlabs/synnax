// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	"context"
	"iter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// constService is an ontology.Service whose resources always carry a fixed name, used
// to distinguish which of several registered services answered a retrieval.
type constService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	typ  resource.Type
	name string
}

func (s constService) Type() resource.Type { return s.typ }

func (s constService) RetrieveResource(
	_ context.Context, key string, _ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.NewResource(
		schema, resource.ID{Type: s.typ, Key: key}, s.name, Sample{Key: key},
	), nil
}

func (constService) Search(
	context.Context,
	gorp.Tx,
	search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return search.Result[resource.ID]{}, nil
}

var _ = Describe("Service", func() {
	It(
		"Should panic when retrieving a resource whose service is not registered",
		func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			id := resource.ID{Type: resource.TypeUser, Key: "unregistered"}
			Expect(otg.NewWriter(tx).DefineResources(ctx, id)).To(Succeed())
			var res ontology.Resource
			Expect(func() {
				Expect(
					otg.NewRetrieve().WhereIDs(id).Entry(&res).Exec(ctx, tx),
				).To(Succeed())
			}).To(PanicWith(ContainSubstring("service user not found")))
		},
	)

	It(
		"Should keep the first service when a duplicate type is registered",
		func(ctx SpecContext) {
			d := DeferClose(gorp.Wrap(memkv.New()))
			o := MustOpen(ontology.Open(ctx, ontology.Config{DB: d}))
			o.RegisterService(
				constService{typ: resource.TypeChannel, name: "first"},
			)
			o.RegisterService(
				constService{typ: resource.TypeChannel, name: "second"},
			)
			id := resource.ID{Type: resource.TypeChannel, Key: "dup"}
			Expect(o.NewWriter(nil).DefineResources(ctx, id)).To(Succeed())
			var res ontology.Resource
			Expect(
				o.NewRetrieve().WhereIDs(id).Entry(&res).Exec(ctx, nil),
			).To(Succeed())
			Expect(res.Name).To(Equal("first"))
		},
	)
})
