// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"context"
	"io"
	"iter"
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

func TestLibrary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Library Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// namedService serves resources of one type that carry only a name.
type namedService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	t     ontology.ResourceType
	names map[string]string
}

var _ ontology.Service = (*namedService)(nil)

var nameSchema = zyn.Object(map[string]zyn.Schema{"name": zyn.String()})

func (s *namedService) Type() ontology.ResourceType { return s.t }

func (s *namedService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	name, ok := s.names[key]
	if !ok {
		return ontology.Resource{}, query.ErrNotFound
	}
	return ontology.NewResource(
		nameSchema,
		ontology.ID{Type: s.t, Key: key},
		name,
		map[string]any{"name": name},
	), nil
}

func (s *namedService) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	resources := lo.Map(
		lo.Keys(s.names),
		func(key string, _ int) ontology.Resource {
			return lo.Must(s.RetrieveResource(ctx, key, nil))
		},
	)
	return slices.Values(resources), xio.NopCloser, nil
}

var (
	db    *gorp.DB
	otg   *ontology.Ontology
	tasks *namedService
	docs  *namedService
	tx    gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	tasks = &namedService{t: ontology.ResourceTypeTask, names: map[string]string{}}
	docs = &namedService{t: ontology.ResourceTypeICD, names: map[string]string{}}
	otg.RegisterService(tasks)
	otg.RegisterService(docs)
})

var _ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
