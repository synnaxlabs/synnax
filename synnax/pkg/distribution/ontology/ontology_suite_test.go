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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}

type sampleService struct {
	observe.Noop[iter.Nexter[ontology.Change]]
}

var _ ontology.Service = (*sampleService)(nil)

const sampleType ontology.Type = "sample"

type Sample struct {
	Key string
}

func newSampleType(key string) ontology.ID {
	return ontology.ID{Key: key, Type: sampleType}
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key": zyn.String(),
})

func (s *sampleService) Type() ontology.Type { return sampleType }

func (s *sampleService) Schema() zyn.Schema { return schema }

func (s *sampleService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	return core.NewResource(s.Schema(), newSampleType(key), "empty", Sample{Key: key}), nil
}

func (s *sampleService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser(iter.All([]ontology.Resource{
		lo.Must(s.RetrieveResource(ctx, "", nil)),
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
	otg.RegisterService(&sampleService{})
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
