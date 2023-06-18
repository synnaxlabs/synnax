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
	"context"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"go.uber.org/zap"
)

type Index struct {
	Config
	mapping *mapping.IndexMappingImpl
	idx     bleve.Index
}

type Config struct {
	alamos.Instrumentation
}

var _ config.Config[Config] = Config{}

func (c Config) Validate() error { return nil }

func (c Config) Override(other Config) Config { return c }

func New(configs ...Config) (*Index, error) {
	cfg, err := config.New(Config{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Index{Config: cfg}
	s.mapping = bleve.NewIndexMapping()
	s.idx, err = bleve.NewMemOnly(s.mapping)
	return s, err
}

func (s *Index) OpenTx() Tx { return Tx{idx: s.idx, batch: s.idx.NewBatch()} }

func (s *Index) WithTx(f func(Tx) error) error {
	t := s.OpenTx()
	defer t.Close()
	if err := f(t); err != nil {
		return err
	}
	return t.Commit()
}

func (s *Index) Index(resources []schema.Resource) error {
	t := s.OpenTx()
	defer t.Close()
	for _, r := range resources {
		if err := t.Index(r); err != nil {
			return err
		}
	}
	return t.Commit()
}

type Tx struct {
	idx   bleve.Index
	batch *bleve.Batch
}

func (t *Tx) Apply(changes ...schema.Change) error {
	for _, ch := range changes {
		if ch.Variant == change.Set {
			if err := t.batch.Index(ch.Key.String(), ch.Value); err != nil {
				return err
			}
		} else {
			t.batch.Delete(ch.Key.String())
		}
	}
	return nil
}

func (t *Tx) Commit() error { return t.idx.Batch(t.batch) }

func (t *Tx) Index(resource schema.Resource) error {
	return t.batch.Index(resource.ID.String(), resource)
}

func (t *Tx) Delete(id schema.ID) { t.batch.Delete(id.String()) }

func (t *Tx) Close() { t.idx = nil; t.batch = nil }

func (s *Index) Register(ctx context.Context, sch schema.Schema) {
	s.L.Debug("registering schema", zap.String("type", string(sch.Type)))
	_, span := s.T.Prod(ctx, "register")
	defer span.End()
	dm := bleve.NewDocumentMapping()
	dm.AddFieldMappingsAt("Name", bleve.NewTextFieldMapping())
	for k, f := range sch.Fields {
		dm.AddFieldMappingsAt(k, fieldMappings[f.Type]())
	}
	s.mapping.AddDocumentMapping(string(sch.Type), dm)
}

type Request struct {
	Term string
	Type schema.Type
}

func (s *Index) Search(ctx context.Context, req Request) ([]schema.ID, error) {
	ctx, span := s.T.Prod(ctx, "search")
	q := bleve.NewQueryStringQuery(req.Term)
	search_ := bleve.NewSearchRequest(q)
	search_.Fields = []string{"*"}
	searchResults, err := s.idx.SearchInContext(ctx, search_)
	if err != nil {
		return nil, span.EndWith(err)
	}
	ids, err := schema.ParseIDs(lo.Map(
		searchResults.Hits,
		func(hit *search.DocumentMatch, _ int) string { return hit.ID },
	))
	return ids, span.EndWith(err)
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
