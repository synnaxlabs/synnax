// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestImEx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Import/Export Suite")
}

const (
	testResourceType  ontology.ResourceType = "imex_test"
	errorResourceType ontology.ResourceType = "imex_test_error"
	testVersion       imex.Version          = 1
)

type testResource struct {
	Name     string `json:"name"`
	FieldOne string `json:"field_one"`
	FieldTwo int    `json:"field_two"`
}

func sampleResource(name string) testResource {
	return testResource{Name: name, FieldOne: "value", FieldTwo: 42}
}

func sampleEnvelope(name string, typ ontology.ResourceType) imex.Envelope {
	return MustSucceed(imex.Encode(sampleResource(name), testVersion, typ))
}

type testEntry struct {
	Key      string `json:"key" msgpack:"key"`
	Name     string `json:"name" msgpack:"name"`
	FieldOne string `json:"field_one" msgpack:"field_one"`
	FieldTwo int    `json:"field_two" msgpack:"field_two"`
}

var _ gorp.Entry[string] = testEntry{}

func (e testEntry) GorpKey() string { return e.Key }
func (testEntry) SetOptions() []any { return nil }

// testService is a minimal in-memory ImportExporter used to exercise the ImEx registry
// without depending on any concrete service. Imports decode the typed payload, allocate
// a fresh key, and persist through the provided transaction; exports look up by key and
// re-encode through Encode.
type testService struct {
	db    *gorp.DB
	table *gorp.Table[string, testEntry]
}

func openTestService(ctx context.Context, db *gorp.DB) *testService {
	table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, testEntry]{DB: db}))
	return &testService{db: db, table: table}
}

func (*testService) Type() ontology.ResourceType { return testResourceType }

func (s *testService) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
) (imex.Key, error) {
	r, err := imex.Decode[testResource](ctx, env)
	if err != nil {
		return "", err
	}
	key := uuid.NewString()
	e := testEntry{Key: key, Name: r.Name, FieldOne: r.FieldOne, FieldTwo: r.FieldTwo}
	if err := s.table.NewCreate().Entry(&e).Exec(ctx, tx); err != nil {
		return "", err
	}
	return key, nil
}

func (s *testService) Export(
	ctx context.Context,
	tx gorp.Tx,
	key imex.Key,
) (imex.Envelope, error) {
	var e testEntry
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, testEntry](key)).
		Entry(&e).
		Exec(ctx, tx); err != nil {
		return imex.Envelope{}, err
	}
	return imex.Encode(
		testResource{Name: e.Name, FieldOne: e.FieldOne, FieldTwo: e.FieldTwo},
		testVersion,
		testResourceType,
	)
}

func (s *testService) Retrieve(ctx context.Context, name string) (testEntry, error) {
	var e testEntry
	if err := s.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, e *testEntry) (bool, error) {
			return e.Name == name, nil
		})).
		Entry(&e).
		Exec(ctx, s.db); err != nil {
		return testEntry{}, err
	}
	return e, nil
}

func (s *testService) Close() error { return s.table.Close() }

// errorService is an ImportExporter whose Import and Export methods always return an
// error. It exists to verify that the registry surfaces handler-side failures verbatim.
type errorService struct{}

func (errorService) Type() ontology.ResourceType { return errorResourceType }

func (errorService) Import(context.Context, gorp.Tx, imex.Envelope) (imex.Key, error) {
	return "", errors.New("importer error: forced failure")
}

func (errorService) Export(context.Context, gorp.Tx, imex.Key) (imex.Envelope, error) {
	return imex.Envelope{}, errors.New("exporter error: forced failure")
}

// noopImporter is a minimal Importer that always succeeds without persisting anything.
// Its Type is configurable so callers can exercise asymmetric registration.
type noopImporter struct{ typ ontology.ResourceType }

func (n noopImporter) Type() ontology.ResourceType { return n.typ }

func (noopImporter) Import(context.Context, gorp.Tx, imex.Envelope) (imex.Key, error) {
	return "noop-key", nil
}

// noopExporter is a minimal Exporter that returns a fixed envelope under its Type.
type noopExporter struct{ typ ontology.ResourceType }

func (n noopExporter) Type() ontology.ResourceType { return n.typ }

func (n noopExporter) Export(
	context.Context,
	gorp.Tx,
	imex.Key,
) (imex.Envelope, error) {
	return imex.Encode(testResource{Name: "noop"}, testVersion, n.typ)
}

var (
	db  *gorp.DB
	svc *imex.Service
	ts  *testService
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = DeferClose(gorp.Wrap(memkv.New()))
	svc = MustSucceed(imex.NewService(imex.ServiceConfig{DB: db}))
	ts = DeferClose(openTestService(ctx, db))
	svc.RegisterImportExporter(ts)
	svc.RegisterImportExporter(errorService{})
})
