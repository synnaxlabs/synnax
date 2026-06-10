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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

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
	env := imex.Envelope{Version: testVersion, Type: string(typ)}
	Expect(imex.Encode(&env, sampleResource(name))).To(Succeed())
	return wireRoundTrip(env)
}

func wireRoundTrip(env imex.Envelope) imex.Envelope {
	b := MustSucceed(json.Marshal(env))
	var out imex.Envelope
	Expect(json.Unmarshal(b, &out)).To(Succeed())
	return out
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
	table := MustSucceed(
		gorp.OpenTable(ctx, gorp.TableConfig[string, testEntry]{DB: db}),
	)
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

func (s *testService) Export(ctx context.Context, key imex.Key) (imex.Envelope, error) {
	var e testEntry
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, testEntry](key)).
		Entry(&e).
		Exec(ctx, s.db); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: testVersion, Type: string(testResourceType)}
	if err := imex.Encode(
		&env,
		testResource{Name: e.Name, FieldOne: e.FieldOne, FieldTwo: e.FieldTwo},
	); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
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

type errorService struct{}

func (errorService) Type() ontology.ResourceType { return errorResourceType }

func (errorService) Import(context.Context, gorp.Tx, imex.Envelope) (imex.Key, error) {
	return "", errors.New("importer error: forced failure")
}

func (errorService) Export(context.Context, imex.Key) (imex.Envelope, error) {
	return imex.Envelope{}, errors.New("exporter error: forced failure")
}

type noopImporter struct{ typ ontology.ResourceType }

func (n noopImporter) Type() ontology.ResourceType { return n.typ }

func (noopImporter) Import(context.Context, gorp.Tx, imex.Envelope) (imex.Key, error) {
	return "noop-key", nil
}

type noopExporter struct{ typ ontology.ResourceType }

func (n noopExporter) Type() ontology.ResourceType { return n.typ }

func (n noopExporter) Export(context.Context, imex.Key) (imex.Envelope, error) {
	env := imex.Envelope{Version: testVersion, Type: string(n.typ)}
	if err := imex.Encode(&env, testResource{Name: "noop"}); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

var _ = Describe("Service", func() {
	var (
		svc *imex.Service
		ts  *testService
	)
	BeforeEach(func(ctx SpecContext) {
		svc = imex.NewService()
		ts = DeferClose(openTestService(ctx, db))
		svc.RegisterImportExporter(ts)
		svc.RegisterImportExporter(errorService{})
	})

	Describe("ImporterType", func() {
		It("Should return the registered importer's broader Type", func() {
			Expect(
				svc.ImporterType(string(testResourceType)),
			).To(Equal(testResourceType))
		})

		It("Should return a validation error scoped to the type field if not registered", func() {
			Expect(svc.ImporterType("nonexistent")).Error().To(SatisfyAll(
				MatchError(ContainSubstring("type")),
				MatchError(ContainSubstring("no importer registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
	})

	Describe("RegisterImporter", func() {
		It("Should register an importer under a narrow type string", func() {
			s := imex.NewService()
			s.RegisterImporter("narrow", noopImporter{typ: "broad"})
			Expect(s.ImporterType("narrow")).To(Equal(ontology.ResourceType("broad")))
		})

		It(
			"Should map the narrow type to the importer's broader Type for access control",
			func(ctx SpecContext) {
				s := imex.NewService()
				s.RegisterImporter("http_read", noopImporter{typ: "task"})
				s.RegisterImporter("opc_scan", noopImporter{typ: "task"})
				Expect(s.ImporterType("http_read")).To(Equal(ontology.ResourceType("task")))
				Expect(s.ImporterType("opc_scan")).To(Equal(ontology.ResourceType("task")))
				k1 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "http_read", Name: "ingest"},
				))
				k2 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "opc_scan", Name: "scan"},
				))
				Expect(k1).To(Equal("noop-key"))
				Expect(k2).To(Equal("noop-key"))
			},
		)
	})

	Describe("RegisterExporter", func() {
		It("Should register an exporter under its own Type", func(ctx SpecContext) {
			s := imex.NewService()
			s.RegisterExporter(noopExporter{typ: "noop_export"})
			env := MustSucceed(s.Export(ctx, ontology.ID{
				Type: "noop_export",
				Key:  "any",
			}))
			Expect(env.Type).To(Equal("noop_export"))
			Expect(env.Name).To(Equal("noop"))
		})
	})

	Describe("Import", func() {
		It("Should route to the correct service by type and return the new key", func(ctx SpecContext) {
			key := MustSucceed(svc.Import(
				ctx, db, sampleEnvelope("Registry Test", testResourceType),
			))
			Expect(key).NotTo(BeEmpty())
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			env := imex.Envelope{
				Version: testVersion,
				Type:    "nonexistent",
				Name:    "Bad Type",
			}
			Expect(svc.Import(ctx, db, env)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("no importer registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
		It("Should pass errors from the importer through verbatim", func(ctx SpecContext) {
			Expect(svc.Import(
				ctx, db, sampleEnvelope("Erroring", errorResourceType),
			)).Error().To(MatchError(ContainSubstring("importer error: forced failure")))
		})

		It("Should roll back the transaction when a sibling envelope fails", func(ctx SpecContext) {
			// Per-envelope atomicity is now the caller's responsibility —
			// Service.Import takes a single envelope on a single Tx, and the caller
			// wraps the multi- envelope batch in db.WithTx. This test exercises that
			// contract: the bad envelope causes the tx callback to return early, so the
			// good envelope's write is rolled back along with it.
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				if _, err := svc.Import(
					ctx, tx, sampleEnvelope("Good Record", testResourceType),
				); err != nil {
					return err
				}
				_, err := svc.Import(ctx, tx, imex.Envelope{
					Version: testVersion,
					Type:    "nonexistent",
					Name:    "Bad Type",
				})
				return err
			})
			Expect(err).To(MatchError(ContainSubstring("no importer registered")))
			Expect(ts.Retrieve(ctx, "Good Record")).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
	})

	Describe("Export", func() {
		It("Should round-trip a registered resource through Import then Export", func(ctx SpecContext) {
			key := MustSucceed(svc.Import(
				ctx, db, sampleEnvelope("Round Trip", testResourceType),
			))
			env := MustSucceed(svc.Export(ctx, ontology.ID{
				Type: testResourceType,
				Key:  key,
			}))
			Expect(env.Version).To(Equal(testVersion))
			Expect(env.Type).To(Equal(string(testResourceType)))
			Expect(env.Name).To(Equal("Round Trip"))
			roundTripped := MustSucceed(imex.Decode[testResource](ctx, wireRoundTrip(env)))
			Expect(roundTripped.FieldOne).To(Equal("value"))
			Expect(roundTripped.FieldTwo).To(Equal(42))
		})
		It("Should pass errors from the exporter through verbatim", func(ctx SpecContext) {
			Expect(svc.Export(ctx, ontology.ID{
				Type: errorResourceType,
				Key:  "any-key",
			})).Error().To(MatchError(ContainSubstring("exporter error: forced failure")))
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			Expect(svc.Export(ctx, ontology.ID{
				Type: "nonexistent",
				Key:  "660e8400-e29b-41d4-a716-446655440000",
			})).Error().To(SatisfyAll(
				MatchError(ContainSubstring("no exporter registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
	})

	Describe("Concurrency", func() {
		It("Should be safe to register and look up handlers from multiple goroutines", func() {
			s := imex.NewService()
			const N = 64
			types := make([]string, N)
			for i := range types {
				types[i] = fmt.Sprintf("type-%d", i)
			}
			var wg sync.WaitGroup
			wg.Add(N)
			for _, t := range types {
				go func(t string) {
					defer wg.Done()
					s.RegisterImporter(t, noopImporter{typ: "task"})
				}(t)
			}
			wg.Wait()
			wg.Add(N)
			for _, t := range types {
				go func(t string) {
					defer wg.Done()
					Expect(s.ImporterType(t)).To(Equal(ontology.ResourceType("task")))
				}(t)
			}
			wg.Wait()
		})
	})
})
