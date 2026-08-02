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
	"io"
	"iter"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

// constService is an ontology.Service whose resources always carry a fixed name, used
// to distinguish which of several registered services answered a retrieval.
type constService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	typ  ontology.ResourceType
	name string
}

func (s constService) Type() ontology.ResourceType { return s.typ }

func (s constService) RetrieveResource(
	_ context.Context, key string, _ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.NewResource(
		schema, ontology.ID{Type: s.typ, Key: key}, s.name, Sample{Key: key},
	), nil
}

func (constService) OpenNexter(
	context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values([]ontology.Resource{}), xio.NopCloser, nil
}

var _ = Describe("Service", func() {
	It("Should panic when retrieving a resource whose service is not registered", func(ctx SpecContext) {
		id := ontology.ID{Type: ontology.ResourceTypeUser, Key: "unregistered"}
		Expect(otg.NewWriter(tx).DefineResources(ctx, id)).To(Succeed())
		var res ontology.Resource
		Expect(func() {
			_ = otg.NewRetrieve().WhereIDs(id).Entry(&res).Exec(ctx, tx)
		}).To(PanicWith(ContainSubstring("service user not found")))
	})

	It("Should keep the first service when a duplicate type is registered", func(ctx SpecContext) {
		d := DeferClose(gorp.Wrap(memkv.New()))
		o := MustOpen(ontology.Open(ctx, ontology.Config{DB: d}))
		o.RegisterService(constService{typ: ontology.ResourceTypeChannel, name: "first"})
		o.RegisterService(constService{typ: ontology.ResourceTypeChannel, name: "second"})
		id := ontology.ID{Type: ontology.ResourceTypeChannel, Key: "dup"}
		Expect(o.NewWriter(nil).DefineResources(ctx, id)).To(Succeed())
		var res ontology.Resource
		Expect(o.NewRetrieve().WhereIDs(id).Entry(&res).Exec(ctx, nil)).To(Succeed())
		Expect(res.Name).To(Equal("first"))
	})
})
