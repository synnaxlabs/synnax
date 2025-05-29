// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
)

// mockIndexingService implements the Service interface for testing startup indexing
type mockIndexingService struct {
	observe.Observer[iter.Nexter[schema.Change]]
	resources []Resource
	schema_   *Schema
}

var _ Service = (*mockIndexingService)(nil)

func newMockIndexingService(schema_ *Schema, resources []Resource) *mockIndexingService {
	return &mockIndexingService{
		Observer:  observe.New[iter.Nexter[schema.Change]](),
		resources: resources,
		schema_:   schema_,
	}
}

func (s *mockIndexingService) Schema() *Schema {
	return s.schema_
}

func (s *mockIndexingService) OpenNexter() (iter.NexterCloser[Resource], error) {
	return iter.NexterNopCloser[Resource](iter.All[Resource](s.resources)), nil
}

func (s *mockIndexingService) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (Resource, error) {
	for _, r := range s.resources {
		if r.ID.Key == key {
			return r, nil
		}
	}
	return Resource{}, nil
}

var _ = Describe("Search Indexing", func() {
	var (
		ctx       = context.Background()
		ontology_ *Ontology
		db        *gorp.DB
		mockSvc   *mockIndexingService
	)

	const testType Type = "test-type"

	BeforeEach(func() {
		var err error
		db = gorp.Wrap(memkv.New())
		ontology_, err = Open(ctx, Config{
			Instrumentation: alamos.New("test"),
			DB:              db,
			EnableSearch:    config.True(),
		})
		Expect(err).NotTo(HaveOccurred())

		// Create a schema with searchable fields
		schema_ := &Schema{
			Type:   testType,
			Fields: map[string]schema.Field{},
		}

		// Create test resources
		resources := []Resource{
			{
				ID:     ID{Type: testType, Key: "1"},
				Name:   "cat",
				Schema: schema_,
				Data:   map[string]any{},
			},
			{
				ID:     ID{Type: testType, Key: "2"},
				Name:   "Test Resource Two",
				Schema: schema_,
				Data:   map[string]any{},
			},
			{
				ID:     ID{Type: testType, Key: "3"},
				Name:   "Special_Resource_Three",
				Schema: schema_,
				Data:   map[string]any{},
			},
			{
				ID:     ID{Type: testType, Key: "4"},
				Name:   "UPPERCASE RESOURCE",
				Schema: schema_,
				Data:   map[string]any{},
			},
		}

		// Create and register the mock service
		mockSvc = newMockIndexingService(schema_, resources)
		tx := db.OpenTx()
		w := ontology_.NewWriter(tx)
		for _, r := range resources {
			Expect(w.DefineResource(ctx, r.ID)).To(Succeed())
		}
		Expect(tx.Commit(ctx)).To(Succeed())
		ontology_.RegisterService(ctx, mockSvc)
	})

	AfterEach(func() {
		Expect(ontology_.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	It("should index all resources during startup", func() {
		// Run the startup indexing
		err := ontology_.RunStartupSearchIndexing(ctx)
		Expect(err).NotTo(HaveOccurred())

		// Test exact name search
		results, err := ontology_.Search(ctx, search.Request{
			Type: testType,
			Term: "cat",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].ID.Key).To(Equal("1"))
	})
})
