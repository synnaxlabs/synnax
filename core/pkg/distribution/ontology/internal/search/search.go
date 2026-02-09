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
	"go.uber.org/zap"
)

type Index struct {
	idx        bleve.Index
	mapping    *mapping.IndexMappingImpl
	fields     []string
	typeFields map[string][]string
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
	s := &Index{Config: cfg, typeFields: make(map[string][]string)}
	s.mapping = bleve.NewIndexMapping()
	// Don't auto-discover and index unmapped fields via reflection.
	s.mapping.DefaultMapping.Dynamic = false
	// Don't store unmapped field values (we only use hit.ID, never field contents).
	s.mapping.StoreDynamic = false
	// Don't build search indexes for unmapped fields.
	s.mapping.IndexDynamic = false
	// Don't generate columnar doc-values for unmapped fields.
	s.mapping.DocValuesDynamic = false
	if err = registerSeparatorAnalyzer(s.mapping); err != nil {
		return nil, err
	}
	if s.idx, err = bleve.NewMemOnly(s.mapping); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Index) OpenTx() Tx {
	return Tx{idx: s.idx, batch: s.idx.NewBatch(), buf: make(bleveDoc, 8)}
}

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
	buf   bleveDoc
}

// bleveDoc is a flat map indexed by bleve. It merges Resource.Name and
// Resource.Data fields into a single level so that field mappings like "make"
// resolve directly instead of requiring a "data.make" sub-document path.
type bleveDoc map[string]any

func (d bleveDoc) BleveType() string { return d["_type"].(string) }

func (t *Tx) flatten(r resource.Resource) bleveDoc {
	clear(t.buf)
	t.buf["name"] = r.Name
	t.buf["_type"] = r.ID.Type.String()
	if data, ok := r.Data.(map[string]any); ok {
		for k, v := range data {
			t.buf[k] = v
		}
	}
	return t.buf
}

func (t *Tx) Apply(changes ...resource.Change) error {
	for _, ch := range changes {
		if ch.Variant == change.VariantSet {
			if err := t.batch.Index(ch.Key.String(), t.flatten(ch.Value)); err != nil {
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
	return t.batch.Index(resource.ID.String(), t.flatten(resource))
}

func (t *Tx) Delete(id resource.ID) { t.batch.Delete(id.String()) }

func (t *Tx) Close() { t.idx = nil; t.batch = nil }

func (s *Index) Register(
	ctx context.Context,
	t resource.Type,
	searchableFields ...string,
) {
	s.L.Debug("registering schema", zap.Stringer("type", t))
	_, span := s.T.Prod(ctx, "register")
	defer span.End()
	dm := bleve.NewDocumentMapping()
	dm.Dynamic = false
	// Disable _all composite field — we use field-specific queries instead,
	// which gives correct per-field TF-IDF scoring.
	allMapping := bleve.NewDocumentMapping()
	allMapping.Enabled = false
	dm.AddSubDocumentMapping("_all", allMapping)
	allFields := append([]string{"name"}, searchableFields...)
	for _, field := range allFields {
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = separatorAnalyzer
		fm.Store = false
		fm.IncludeTermVectors = false
		fm.DocValues = false
		dm.AddFieldMappingsAt(field, fm)
		if !lo.Contains(s.fields, field) {
			s.fields = append(s.fields, field)
		}
	}
	s.typeFields[t.String()] = allFields
	s.mapping.AddDocumentMapping(t.String(), dm)
}

type Request struct {
	Term string
	Type resource.Type
}

func assembleWordQuery(word string, fields []string) query.Query {
	queries := make([]query.Query, 0, len(fields)*3)
	for _, field := range fields {
		exactQ := bleve.NewMatchQuery(word)
		exactQ.SetFuzziness(0)
		exactQ.SetBoost(100)
		exactQ.SetField(field)
		prefixQ := bleve.NewPrefixQuery(word)
		prefixQ.SetField(field)
		fuzzyQ := bleve.NewMatchQuery(word)
		fuzzyQ.SetFuzziness(1)
		fuzzyQ.SetField(field)
		queries = append(queries, exactQ, prefixQ, fuzzyQ)
	}
	return bleve.NewDisjunctionQuery(queries...)
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
	fields := s.fields
	if len(req.Type) > 0 {
		if tf, ok := s.typeFields[req.Type.String()]; ok {
			fields = tf
		}
	}
	words := strings.FieldsFunc(req.Term, func(r rune) bool { return r == ' ' || r == '_' || r == '-' })
	wordQueries := make([]query.Query, len(words))
	for i, word := range words {
		wordQueries[i] = assembleWordQuery(word, fields)
	}
	cj := bleve.NewConjunctionQuery(wordQueries...)
	res, err := s.execQuery(ctx, cj)
	if err != nil {
		return nil, span.EndWith(err)
	}
	// If there are no results, reuse the same word queries as a disjunction fallback.
	if res.Total == 0 {
		dq := bleve.NewDisjunctionQuery(wordQueries...)
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
