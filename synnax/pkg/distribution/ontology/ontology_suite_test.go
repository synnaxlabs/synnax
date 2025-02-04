// Copyright 2025 Synnax Labs, Inc.
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
	"testing"
)

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}

type emptyService struct {
	observe.Noop[iter.Nexter[schema.Change]]
}

var _ ontology.Service = (*emptyService)(nil)

const emptyType ontology.Type = "empty"

func newEmptyID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: emptyType}
}

func (s *emptyService) Schema() *ontology.Schema {
	return &ontology.Schema{
		Type: emptyType,
		Fields: map[string]schema.Field{
			"key": {Type: schema.String},
		},
	}
}

func (s *emptyService) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	e := schema.NewResource(s.Schema(), newEmptyID(key), "empty")
	schema.Set(e, "key", key)
	return e, nil
}

func (s *emptyService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser(iter.All([]ontology.Resource{
		schema.NewResource(s.Schema(), newEmptyID(""), "empty"),
	})), nil
}

var (
	ctx = context.Background()
	db  *gorp.DB
	otg *ontology.Ontology
	tx  gorp.Tx
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	otg.RegisterService(&emptyService{})
})

var _ = AfterSuite(func() {
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	tx = db.OpenTx()
})

var _ = AfterEach(func() {
	Expect(tx.Close()).To(Succeed())
})
