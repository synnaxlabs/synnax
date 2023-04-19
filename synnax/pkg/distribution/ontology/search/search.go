// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

type Index struct {
	mapping *mapping.IndexMappingImpl
	index   bleve.Index
}

func New() (*Index, error) {
	s := &Index{}
	s.mapping = bleve.NewIndexMapping()
	var err error
	s.index, err = bleve.NewMemOnly(s.mapping)
	return s, err
}

func (s *Index) Index(resources []schema.Resource) error {
	b := s.index.NewBatch()
	for _, e := range resources {
		if err := b.Index(e.ID.String(), e); err != nil {
			return err
		}
	}
	return s.index.Batch(b)
}

func (s *Index) Register(sch schema.Schema) {
	dm := bleve.NewDocumentMapping()
	dm.AddFieldMappingsAt("Name", bleve.NewTextFieldMapping())
	for k, f := range sch.Fields {
		dm.AddFieldMappingsAt(k, fieldMappings[f.Type]())
	}
	s.mapping.AddDocumentMapping(sch.Type, dm)
}

func (s *Index) Search(term string) ([]schema.ID, error) {
	q := bleve.NewQueryStringQuery(term)
	search_ := bleve.NewSearchRequest(q)
	search_.Fields = []string{"*"}
	searchResults, err := s.index.Search(search_)
	if err != nil {
		return nil, err
	}
	return schema.ParseIDs(lo.Map(
		searchResults.Hits,
		func(hit *search.DocumentMatch, _ int) string { return hit.ID },
	))
}

var fieldMappings = map[schema.FieldType]func() *mapping.FieldMapping{
	schema.String:  bleve.NewTextFieldMapping,
	schema.Int:     bleve.NewNumericFieldMapping,
	schema.Float64: bleve.NewNumericFieldMapping,
	schema.Float32: bleve.NewNumericFieldMapping,
	schema.Int64:   bleve.NewNumericFieldMapping,
	schema.Int32:   bleve.NewNumericFieldMapping,
	schema.Int16:   bleve.NewNumericFieldMapping,
	schema.Int8:    bleve.NewNumericFieldMapping,
	schema.Uint64:  bleve.NewNumericFieldMapping,
	schema.Uint32:  bleve.NewNumericFieldMapping,
	schema.Uint16:  bleve.NewNumericFieldMapping,
	schema.Uint8:   bleve.NewNumericFieldMapping,
	schema.Bool:    bleve.NewBooleanFieldMapping,
	schema.UUID:    bleve.NewTextFieldMapping,
}
