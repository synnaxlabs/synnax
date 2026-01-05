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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// mockIndexingService implements the Service interface for testing startup indexing
type mockIndexingService struct {
	observe.Observer[iter.Seq[ontology.Change]]
	resources []ontology.Resource
	schema    zyn.Schema
}

var _ ontology.Service = (*mockIndexingService)(nil)

func newMockIndexingService(schema zyn.Schema, resources []ontology.Resource) *mockIndexingService {
	return &mockIndexingService{
		Observer:  observe.New[iter.Seq[ontology.Change]](),
		resources: resources,
		schema:    schema,
	}
}

const testType ontology.Type = "test-type"

func (s *mockIndexingService) Type() ontology.Type { return testType }

func (s *mockIndexingService) Schema() zyn.Schema {
	return s.schema
}

func (s *mockIndexingService) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values(s.resources), xio.NopCloser, nil
}

func (s *mockIndexingService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	for _, r := range s.resources {
		if r.ID.Key == key {
			return r, nil
		}
	}
	return ontology.Resource{}, nil
}

var _ = Describe("SearchTerm Indexing", func() {
	var (
		mockSvc *mockIndexingService
	)

	BeforeEach(func() {
		z := zyn.Object(nil)
		resources := []ontology.Resource{
			ontology.NewResource(
				z,
				ontology.ID{Type: testType, Key: "1"},
				"cat",
				map[string]any{},
			),
			ontology.NewResource(
				z,
				ontology.ID{Type: testType, Key: "2"},
				"Test Resource Two",
				map[string]any{},
			),
			ontology.NewResource(
				z,
				ontology.ID{Type: testType, Key: "3"},
				"Special_Resource_Three",
				map[string]any{},
			),
			ontology.NewResource(
				z,
				ontology.ID{Type: testType, Key: "4"},
				"UPPERCASE RESOURCE",
				map[string]any{},
			),
		}

		// Create and register the mock service
		mockSvc = newMockIndexingService(z, resources)
		Expect(otg.InitializeSearchIndex(ctx)).To(Succeed())
		tx := db.OpenTx()
		w := otg.NewWriter(tx)
		for _, r := range resources {
			Expect(w.DefineResource(ctx, r.ID)).To(Succeed())
		}
		Expect(tx.Commit(ctx)).To(Succeed())
		otg.RegisterService(mockSvc)
	})
})
