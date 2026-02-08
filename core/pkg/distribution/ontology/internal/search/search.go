// Copyright 2026 Synnax Labs, Inc.
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
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/resource"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/zyn"
	"go.uber.org/zap"
)

type Index struct {
	idx     bleve.Index
	mapping *mapping.IndexMappingImpl
	Config
}

type Config struct{ alamos.Instrumentation }

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
	s.mapping.DefaultMapping.Dynamic = false
	s.mapping.StoreDynamic = false
	s.mapping.IndexDynamic = false
	s.mapping.DocValuesDynamic = false
	if err = registerSeparatorAnalyzer(s.mapping); err != nil {
		return nil, err
	}
	if s.idx, err = bleve.NewMemOnly(s.mapping); err != nil {
		return nil, err
	}
	return s, nil
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

func (s *Index) Index(resources []resource.Resource) error {
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

func (t *Tx) Apply(changes ...resource.Change) error {
	for _, ch := range changes {
		if ch.Variant == change.VariantSet {
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

func (t *Tx) Index(resource resource.Resource) error {
	return t.batch.Index(resource.ID.String(), resource)
}

func (t *Tx) Delete(id resource.ID) { t.batch.Delete(id.String()) }

func (t *Tx) Close() { t.idx = nil; t.batch = nil }

func (s *Index) Register(
	ctx context.Context,
	t resource.Type,
	sch zyn.Schema,
	searchableFields ...string,
) {
	s.L.Debug("registering schema", zap.Stringer("type", t))
	_, span := s.T.Prod(ctx, "register")
	defer span.End()
	dm := bleve.NewDocumentMapping()
	dm.Dynamic = false
	for _, field := range append([]string{"name"}, searchableFields...) {
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = separatorAnalyzer
		fm.Store = false
		fm.IncludeTermVectors = false
		fm.DocValues = false
		dm.AddFieldMappingsAt(field, fm)
	}
	s.mapping.AddDocumentMapping(t.String(), dm)
}

type Request struct {
	Term string
	Type resource.Type
}

func assembleWordQuery(word string, _ int) query.Query {
	fuzzyQ := bleve.NewMatchQuery(word)
	// Specifies the levenshtein distance for the fuzzy query
	// https://en.wikipedia.org/wiki/Levenshtein_distance
	fuzzyQ.SetFuzziness(1)
	prefixQ := bleve.NewPrefixQuery(word)
	exactQ := bleve.NewMatchQuery(word)
	// Specifies the levenshtein distance for the fuzzy query
	// https://en.wikipedia.org/wiki/Levenshtein_distance
	exactQ.SetFuzziness(0)
	// Makes the exact result the most important. Value chosen arbitrarily.
	exactQ.SetBoost(100)
	return bleve.NewDisjunctionQuery(exactQ, prefixQ, fuzzyQ)
}

func (s *Index) execQuery(
	ctx context.Context,
	q query.Query,
) (*bleve.SearchResult, error) {
	req := bleve.NewSearchRequest(q)
	// Limit search results to 100
	req.Size = 100
	req.SortBy([]string{"-_score"})
	return s.idx.SearchInContext(ctx, req)
}

func (s *Index) Search(ctx context.Context, req Request) ([]resource.ID, error) {
	ctx, span := s.T.Prod(ctx, "search")
	words := strings.FieldsFunc(req.Term, func(r rune) bool { return r == ' ' || r == '_' || r == '-' })
	querySet := lo.Map(words, assembleWordQuery)
	cj := bleve.NewConjunctionQuery(querySet...)
	res, err := s.execQuery(ctx, cj)
	if err != nil {
		return nil, span.EndWith(err)
	}
	// If there are no results, fallback to a disjunction query which is more lenient
	if res.Total == 0 {
		dq := bleve.NewDisjunctionQuery(lo.Map(words, assembleWordQuery)...)
		res, err = s.execQuery(ctx, dq)
		if err != nil {
			return nil, span.EndWith(err)
		}
	}
	ids, err := resource.ParseIDs(lo.Map(
		res.Hits,
		func(hit *search.DocumentMatch, _ int) string { return hit.ID },
	))
	if err != nil {
		return nil, span.EndWith(err)
	}
	if len(req.Type) == 0 {
		return ids, nil
	}
	return lo.Filter(ids, func(id resource.ID, _ int) bool { return id.Type == req.Type }), nil
}
