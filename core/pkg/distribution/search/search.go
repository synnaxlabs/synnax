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
	"io"
	"iter"
	"maps"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// Service represents a service whose resources should be indexed for search.
type Service interface {
	Type() ontology.ResourceType
	observe.Observable[iter.Seq[ontology.Change]]
	OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error)
}

// FieldsProvider is an optional interface that services can implement to declare
// which fields beyond "name" should be indexed for search.
type FieldsProvider interface {
	SearchableFields() []string
}

type Index struct {
	idx                 bleve.Index
	mapping             *mapping.IndexMappingImpl
	fields              []string
	typeFields          map[string][]string
	services            []Service
	disconnectObservers []observe.Disconnect
	Config
}

type Config struct {
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

func (c Config) Validate() error { return nil }

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

var ErrNotEnabled = errors.New("[search] - search is not enabled")

func New(configs ...Config) (*Index, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Index{Config: cfg, typeFields: make(map[string][]string)}
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

// RegisterService registers a service for search indexing. The service's resources
// will be indexed when InitializeIndex is called.
func (s *Index) RegisterService(svc Service) {
	if s == nil {
		return
	}
	s.services = append(s.services, svc)
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

func (s *Index) IndexResources(resources []ontology.Resource) error {
	t := s.OpenTx()
	defer t.Close()
	for _, r := range resources {
		if err := t.Index(r); err != nil {
			return err
		}
	}
	return t.Commit()
}

// InitializeIndex registers search schemas for all registered services, indexes
// all existing resources, and hooks OnChange for live sync. This method should be
// called AFTER all services have been registered via RegisterService. It blocks
// until all resources have been indexed.
func (s *Index) InitializeIndex(ctx context.Context) error {
	if s == nil {
		return nil
	}
	for _, svc := range s.services {
		var extraFields []string
		if provider, ok := svc.(FieldsProvider); ok {
			extraFields = provider.SearchableFields()
		}
		s.register(ctx, svc.Type(), extraFields...)
	}
	oCtx, cancel := signal.WithCancel(ctx)
	defer cancel()
	for _, svc := range s.services {
		disconnect := svc.OnChange(func(ctx context.Context, i iter.Seq[ontology.Change]) {
			err := s.WithTx(func(tx Tx) error {
				for ch := range i {
					s.L.Debug(
						"updating search index",
						zap.String("key", ch.Key),
						zap.Stringer("type", svc.Type()),
						zap.Stringer("variant", ch.Variant),
					)
					if err := tx.Apply(ch); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				s.L.Error("failed to index resource",
					zap.Stringer("type", svc.Type()),
					zap.Error(err),
				)
			}
		})
		s.disconnectObservers = append(s.disconnectObservers, disconnect)
		oCtx.Go(func(ctx context.Context) (err error) {
			n, closer, err := svc.OpenNexter(ctx)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Combine(err, closer.Close())
			}()
			err = s.WithTx(func(tx Tx) error {
				for r := range n {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err = tx.Index(r); err != nil {
						return err
					}
				}
				return nil
			})
			return err
		}, signal.WithKeyf("startup_indexing_%s", svc.Type()))
	}
	return oCtx.Wait()
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

func (t *Tx) flatten(r ontology.Resource) bleveDoc {
	clear(t.buf)
	t.buf["name"] = r.Name
	t.buf["_type"] = r.ID.Type.String()
	if data, ok := r.Data.(map[string]any); ok {
		maps.Copy(t.buf, data)
	}
	return t.buf
}

func (t *Tx) Apply(changes ...ontology.Change) error {
	for _, ch := range changes {
		if ch.Variant == change.VariantSet {
			if err := t.batch.Index(ch.Key, t.flatten(ch.Value)); err != nil {
				return err
			}
		} else {
			t.batch.Delete(ch.Key)
		}
	}
	return nil
}

func (t *Tx) Commit() error { return t.idx.Batch(t.batch) }

func (t *Tx) Index(resource ontology.Resource) error {
	return t.batch.Index(resource.ID.String(), t.flatten(resource))
}

func (t *Tx) Delete(id ontology.ID) { t.batch.Delete(id.String()) }

func (t *Tx) Close() { t.idx = nil; t.batch = nil }

func (s *Index) register(
	ctx context.Context,
	t ontology.ResourceType,
	searchableFields ...string,
) {
	s.L.Debug("registering schema", zap.Stringer("type", t))
	_, span := s.T.Prod(ctx, "register")
	defer span.End()
	dm := bleve.NewDocumentMapping()
	dm.Dynamic = false
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
	Type ontology.ResourceType
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
	req.Size = 100
	req.SortBy([]string{"-_score"})
	return s.idx.SearchInContext(ctx, req)
}

func (s *Index) Search(ctx context.Context, req Request) ([]ontology.ID, error) {
	if s == nil {
		return nil, ErrNotEnabled
	}
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
	if res.Total == 0 {
		dq := bleve.NewDisjunctionQuery(wordQueries...)
		res, err = s.execQuery(ctx, dq)
		if err != nil {
			return nil, span.EndWith(err)
		}
	}
	ids, err := ontology.ParseIDs(lo.Map(
		res.Hits,
		func(hit *search.DocumentMatch, _ int) string { return hit.ID },
	))
	if err != nil {
		return nil, span.EndWith(err)
	}
	if len(req.Type) == 0 {
		return ids, nil
	}
	return lo.Filter(ids, func(id ontology.ID, _ int) bool { return id.Type == req.Type }), nil
}
