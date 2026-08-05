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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

const testVersion imex.Version = 1

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
	return WireRoundTrip(env)
}

type testEntry struct {
	Key      string `json:"key"       msgpack:"key"`
	Name     string `json:"name"      msgpack:"name"`
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

func (*testService) Type() ontology.ResourceType { return ontology.ResourceTypeChannel }

func (s *testService) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	r, err := imex.Decode[testResource](ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	key := uuid.NewString()
	e := testEntry{Key: key, Name: env.Name, FieldOne: r.FieldOne, FieldTwo: r.FieldTwo}
	if err := s.table.NewCreate().Entry(&e).Exec(ctx, tx); err != nil {
		return ontology.ID{}, err
	}
	id := ontology.ID{Type: ontology.ResourceTypeChannel, Key: key}
	w := otg.NewWriter(tx)
	if err := w.DefineResources(ctx, id); err != nil {
		return ontology.ID{}, err
	}
	if !opts.Parent.IsZero() {
		if err := w.DefineRelationships(
			ctx, opts.Parent, ontology.RelationshipTypeParentOf, id,
		); err != nil {
			return ontology.ID{}, err
		}
	}
	return id, nil
}

func (*testService) Match(map[string]any) bool { return false }

func (s *testService) Export(
	ctx context.Context,
	id ontology.ID,
) (imex.Envelope, error) {
	var e testEntry
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, testEntry](id.Key)).
		Entry(&e).
		Exec(ctx, s.db); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: testVersion,
		Type:    string(ontology.ResourceTypeChannel),
	}
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

func (errorService) Type() ontology.ResourceType { return ontology.ResourceTypeDevice }

func (errorService) Match(map[string]any) bool { return false }

func (errorService) Import(
	context.Context, gorp.Tx, imex.Envelope, imex.ImportOptions,
) (ontology.ID, error) {
	return ontology.ID{}, errors.New("importer error: forced failure")
}

func (errorService) Export(context.Context, ontology.ID) (imex.Envelope, error) {
	return imex.Envelope{}, errors.New("exporter error: forced failure")
}

type noopImporter struct{ typ ontology.ResourceType }

func (n noopImporter) Type() ontology.ResourceType { return n.typ }

func (noopImporter) Match(map[string]any) bool { return false }

func (n noopImporter) Import(
	context.Context, gorp.Tx, imex.Envelope, imex.ImportOptions,
) (ontology.ID, error) {
	return ontology.ID{Type: n.typ, Key: "noop-key"}, nil
}

// matchImporter is a noopImporter that claims typeless envelopes whose body carries
// marker as a top-level key.
type matchImporter struct {
	noopImporter
	marker string
}

func (m matchImporter) Match(body map[string]any) bool {
	_, ok := body[m.marker]
	return ok
}

type noopExporter struct{ typ ontology.ResourceType }

func (n noopExporter) Type() ontology.ResourceType { return n.typ }

func (n noopExporter) Export(context.Context, ontology.ID) (imex.Envelope, error) {
	env := imex.Envelope{Version: testVersion, Type: string(n.typ)}
	if err := imex.Encode(&env, testResource{Name: "noop"}); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// newParent defines a fresh project resource in the ontology and returns its ID so
// imports satisfy the registry's required-parent check.
func newParent(ctx context.Context) ontology.ID {
	id := project.OntologyID(uuid.New())
	Expect(otg.NewWriter(nil).DefineResources(ctx, id)).To(Succeed())
	return id
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
				svc.ImporterType(string(ontology.ResourceTypeChannel)),
			).To(Equal(ontology.ResourceTypeChannel))
		})

		It(
			"Should return a validation error scoped to the type field if not registered",
			func() {
				Expect(svc.ImporterType("nonexistent")).Error().To(SatisfyAll(
					MatchError(ContainSubstring("type")),
					MatchError(ContainSubstring("no importer registered")),
					MatchError(ContainSubstring("validation error")),
				))
			},
		)
	})

	Describe("ResolveType", func() {
		typelessEnvelope := func(payload string) imex.Envelope {
			var env imex.Envelope
			Expect(json.Unmarshal([]byte(payload), &env)).To(Succeed())
			return env
		}

		It("Should return a non-empty envelope type as-is", func(ctx SpecContext) {
			Expect(svc.ResolveType(ctx, imex.Envelope{Type: "anything"})).To(
				Equal("anything"),
			)
		})

		It(
			"Should resolve a typeless envelope through the claiming matcher",
			func(ctx SpecContext) {
				s := imex.NewService()
				s.RegisterImporter("aaa", matchImporter{
					noopImporter{typ: ontology.ResourceTypeLog}, "channels",
				})
				s.RegisterImporter("bbb", matchImporter{
					noopImporter{typ: ontology.ResourceTypeSchematic}, "nodes",
				})
				Expect(s.ResolveType(
					ctx, typelessEnvelope(`{"version":"1.0.0","nodes":[]}`),
				)).To(Equal("bbb"))
			},
		)

		It(
			"Should offer the body to matchers in sorted type order",
			func(ctx SpecContext) {
				s := imex.NewService()
				s.RegisterImporter("bbb", matchImporter{
					noopImporter{typ: ontology.ResourceTypeLog}, "marker",
				})
				s.RegisterImporter("aaa", matchImporter{
					noopImporter{typ: ontology.ResourceTypeLog}, "marker",
				})
				Expect(s.ResolveType(
					ctx, typelessEnvelope(`{"marker":true}`),
				)).To(Equal("aaa"))
			},
		)

		It(
			"Should reject a typeless envelope no matcher claims",
			func(ctx SpecContext) {
				Expect(svc.ResolveType(
					ctx, typelessEnvelope(`{"version":"1.0.0","foo":1}`),
				)).Error().To(SatisfyAll(
					MatchError(
						ContainSubstring("does not match any known resource format"),
					),
					MatchError(ContainSubstring("validation error")),
				))
			},
		)

		It(
			"Should route a typeless envelope through Import via its matcher",
			func(ctx SpecContext) {
				s := imex.NewService()
				s.RegisterImporter("matched", matchImporter{
					noopImporter{typ: ontology.ResourceTypeLog}, "channels",
				})
				id := MustSucceed(s.Import(
					ctx, db, typelessEnvelope(`{"version":"1.0.0","channels":[]}`),
					imex.ImportOptions{FileName: "Legacy.json", Parent: newParent(ctx)},
				))
				Expect(id).To(Equal(ontology.ID{
					Type: ontology.ResourceTypeLog, Key: "noop-key",
				}))
			},
		)
	})

	Describe("RegisterImporter", func() {
		It("Should register an importer under a narrow type string", func() {
			s := imex.NewService()
			s.RegisterImporter(
				"narrow",
				noopImporter{typ: ontology.ResourceTypeChannel},
			)
			Expect(s.ImporterType("narrow")).To(Equal(ontology.ResourceTypeChannel))
		})

		It(
			"Should map the narrow type to the importer's broader Type for access control",
			func(ctx SpecContext) {
				s := imex.NewService()
				s.RegisterImporter(
					"http_read",
					noopImporter{typ: ontology.ResourceTypeTask},
				)
				s.RegisterImporter(
					"opc_scan",
					noopImporter{typ: ontology.ResourceTypeTask},
				)
				Expect(s.ImporterType("http_read")).To(Equal(ontology.ResourceTypeTask))
				Expect(s.ImporterType("opc_scan")).To(Equal(ontology.ResourceTypeTask))
				k1 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "http_read", Name: "ingest"},
					imex.ImportOptions{Parent: newParent(ctx)},
				))
				k2 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "opc_scan", Name: "scan"},
					imex.ImportOptions{Parent: newParent(ctx)},
				))
				Expect(
					k1,
				).To(Equal(ontology.ID{Type: ontology.ResourceTypeTask, Key: "noop-key"}))
				Expect(
					k2,
				).To(Equal(ontology.ID{Type: ontology.ResourceTypeTask, Key: "noop-key"}))
			},
		)
	})

	Describe("RegisterExporter", func() {
		It("Should register an exporter under its own Type", func(ctx SpecContext) {
			s := imex.NewService()
			s.RegisterExporter(noopExporter{typ: ontology.ResourceTypeLog})
			env := MustSucceed(s.Export(ctx, ontology.ID{
				Type: ontology.ResourceTypeLog,
				Key:  "any",
			}))
			Expect(env.Type).To(Equal(string(ontology.ResourceTypeLog)))
			Expect(env.Name).To(Equal("noop"))
		})
	})

	Describe("Import", func() {
		It(
			"Should route to the correct service by type and return the new ID",
			func(ctx SpecContext) {
				id := MustSucceed(svc.Import(
					ctx,
					db,
					sampleEnvelope("Registry Test", ontology.ResourceTypeChannel),
					imex.ImportOptions{Parent: newParent(ctx)},
				))
				Expect(id.Type).To(Equal(ontology.ResourceTypeChannel))
				Expect(id.Key).NotTo(BeEmpty())
			},
		)

		It("Should reject an unregistered type", func(ctx SpecContext) {
			env := imex.Envelope{
				Version: testVersion,
				Type:    "nonexistent",
				Name:    "Bad Type",
			}
			Expect(
				svc.Import(ctx, db, env, imex.ImportOptions{Parent: newParent(ctx)}),
			).Error().
				To(SatisfyAll(
					MatchError(ContainSubstring("no importer registered")),
					MatchError(ContainSubstring("validation error")),
				))
		})
		It(
			"Should pass errors from the importer through verbatim",
			func(ctx SpecContext) {
				Expect(svc.Import(
					ctx, db, sampleEnvelope("Erroring", ontology.ResourceTypeDevice),
					imex.ImportOptions{Parent: newParent(ctx)},
				)).Error().To(MatchError(ContainSubstring("importer error: forced failure")))
			},
		)

		It(
			"Should roll back the transaction when a sibling envelope fails",
			func(ctx SpecContext) {
				// Per-envelope atomicity is now the caller's responsibility —
				// Service.Import takes a single envelope on a single Tx, and the caller
				// wraps the multi- envelope batch in db.WithTx. This test exercises
				// that contract: the bad envelope causes the tx callback to return
				// early, so the good envelope's write is rolled back along with it.
				err := db.WithTx(ctx, func(tx gorp.Tx) error {
					if _, err := svc.Import(
						ctx,
						tx,
						sampleEnvelope("Good Record", ontology.ResourceTypeChannel),
						imex.ImportOptions{Parent: newParent(ctx)},
					); err != nil {
						return err
					}
					_, err := svc.Import(ctx, tx, imex.Envelope{
						Version: testVersion,
						Type:    "nonexistent",
						Name:    "Bad Type",
					}, imex.ImportOptions{Parent: newParent(ctx)})
					return err
				})
				Expect(err).To(MatchError(ContainSubstring("no importer registered")))
				Expect(ts.Retrieve(ctx, "Good Record")).Error().To(
					MatchError(query.ErrNotFound),
				)
			},
		)
	})

	Describe("Import Options", func() {
		namelessEnvelope := func() imex.Envelope {
			b := fmt.Appendf(
				nil,
				`{"version":%d,"type":%q,"field_one":"value","field_two":42}`,
				testVersion, ontology.ResourceTypeChannel,
			)
			var env imex.Envelope
			Expect(json.Unmarshal(b, &env)).To(Succeed())
			return env
		}

		Describe("FileName", func() {
			It(
				"Should fall back to the file name without its extension when the envelope has no name",
				func(ctx SpecContext) {
					id := MustSucceed(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{
							FileName: "Metrics Log.json",
							Parent:   newParent(ctx),
						},
					))
					Expect(id.Key).NotTo(BeEmpty())
					entry := MustSucceed(ts.Retrieve(ctx, "Metrics Log"))
					Expect(entry.Key).To(Equal(id.Key))
				},
			)

			It(
				"Should prefer the envelope's name over the file name",
				func(ctx SpecContext) {
					id := MustSucceed(svc.Import(
						ctx,
						db,
						sampleEnvelope("Body Name", ontology.ResourceTypeChannel),
						imex.ImportOptions{
							FileName: "File Name.json",
							Parent:   newParent(ctx),
						},
					))
					entry := MustSucceed(ts.Retrieve(ctx, "Body Name"))
					Expect(entry.Key).To(Equal(id.Key))
				},
			)

			It(
				"Should reject an envelope with neither a name nor a file name",
				func(ctx SpecContext) {
					Expect(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{Parent: newParent(ctx)},
					)).Error().To(SatisfyAll(
						MatchError(ContainSubstring("name must be a non-empty string")),
						MatchError(ContainSubstring("validation error")),
					))
				},
			)
		})

		Describe("Parent", func() {
			var projectKey project.Key
			BeforeEach(func(ctx SpecContext) {
				projectKey = uuid.New()
				Expect(otg.NewWriter(nil).DefineResources(
					ctx, project.OntologyID(projectKey),
				)).To(Succeed())
			})

			It("Should reject a zero parent", func(ctx SpecContext) {
				Expect(svc.Import(
					ctx, db,
					sampleEnvelope("No Parent", ontology.ResourceTypeChannel),
					imex.ImportOptions{},
				)).Error().To(SatisfyAll(
					MatchError(ContainSubstring("parent")),
					MatchError(ContainSubstring("required")),
				))
			})

			It(
				"Should attach the imported resource under the given parent",
				func(ctx SpecContext) {
					id := MustSucceed(svc.Import(
						ctx,
						db,
						sampleEnvelope("Parented", ontology.ResourceTypeChannel),
						imex.ImportOptions{Parent: project.OntologyID(projectKey)},
					))
					Expect(otg.RelationshipExists(ctx, nil, ontology.Relationship{
						From: project.OntologyID(projectKey),
						Type: ontology.RelationshipTypeParentOf,
						To:   id,
					})).To(BeTrue())
				},
			)

			It(
				"Should roll back the import when the parent does not exist",
				func(ctx SpecContext) {
					err := db.WithTx(ctx, func(tx gorp.Tx) error {
						_, err := svc.Import(
							ctx, tx,
							sampleEnvelope("Orphaned", ontology.ResourceTypeChannel),
							imex.ImportOptions{Parent: project.OntologyID(uuid.New())},
						)
						return err
					})
					Expect(err).To(MatchError(query.ErrNotFound))
					Expect(ts.Retrieve(ctx, "Orphaned")).Error().To(
						MatchError(query.ErrNotFound),
					)
				},
			)
		})
	})

	Describe("Export", func() {
		It(
			"Should round-trip a registered resource through Import then Export",
			func(ctx SpecContext) {
				id := MustSucceed(svc.Import(
					ctx, db, sampleEnvelope("Round Trip", ontology.ResourceTypeChannel),
					imex.ImportOptions{Parent: newParent(ctx)},
				))
				env := MustSucceed(svc.Export(ctx, id))
				Expect(env.Version).To(Equal(testVersion))
				Expect(env.Type).To(Equal(string(ontology.ResourceTypeChannel)))
				Expect(env.Name).To(Equal("Round Trip"))
				roundTripped := MustSucceed(
					imex.Decode[testResource](ctx, WireRoundTrip(env)),
				)
				Expect(roundTripped.FieldOne).To(Equal("value"))
				Expect(roundTripped.FieldTwo).To(Equal(42))
			},
		)
		It(
			"Should pass errors from the exporter through verbatim",
			func(ctx SpecContext) {
				Expect(svc.Export(ctx, ontology.ID{
					Type: ontology.ResourceTypeDevice,
					Key:  "any-key",
				})).Error().To(MatchError(ContainSubstring("exporter error: forced failure")))
			},
		)

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
		It(
			"Should be safe to register and look up handlers from multiple goroutines",
			func() {
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
						s.RegisterImporter(
							t,
							noopImporter{typ: ontology.ResourceTypeTask},
						)
					}(t)
				}
				wg.Wait()
				wg.Add(N)
				for _, t := range types {
					go func(t string) {
						defer wg.Done()
						Expect(s.ImporterType(t)).To(Equal(ontology.ResourceTypeTask))
					}(t)
				}
				wg.Wait()
			},
		)
	})
})
