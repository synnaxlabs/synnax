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
	"encoding/json/v2"
	"fmt"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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

func sampleEnvelope(name string, typ resource.Type) imex.Envelope {
	GinkgoHelper()
	env := imex.Envelope{Version: testVersion, Type: string(typ)}
	Expect(env.Encode(sampleResource(name))).To(Succeed())
	return WireRoundTrip(env)
}

func namelessEnvelope() imex.Envelope {
	GinkgoHelper()
	b := fmt.Appendf(
		nil,
		`{"version":%d,"type":%q,"field_one":"value","field_two":42}`,
		testVersion, resource.TypeChannel,
	)
	var env imex.Envelope
	Expect(json.Unmarshal(b, &env)).To(Succeed())
	return env
}

type testEntry struct {
	Key      string `json:"key"       msgpack:"key"`
	Name     string `json:"name"      msgpack:"name"`
	FieldOne string `json:"field_one" msgpack:"field_one"`
	FieldTwo int    `json:"field_two" msgpack:"field_two"`
	Parent   string `json:"parent"    msgpack:"parent"`
}

var _ gorp.Entry[string] = testEntry{}

func (e testEntry) GorpKey() string { return e.Key }

// EncodeOrc implements orc.SelfEncoder so the entry can persist through the
// production gorp codec, which requires it.
func (e testEntry) EncodeOrc(w *orc.Writer) error {
	w.String(e.Key)
	w.String(e.Name)
	w.String(e.FieldOne)
	w.Int64(int64(e.FieldTwo))
	w.String(e.Parent)
	return nil
}

// DecodeOrc implements orc.SelfDecoder.
func (e *testEntry) DecodeOrc(r *orc.Reader) error {
	var err error
	if e.Key, err = r.String(); err != nil {
		return err
	}
	if e.Name, err = r.String(); err != nil {
		return err
	}
	if e.FieldOne, err = r.String(); err != nil {
		return err
	}
	n, err := r.Int64()
	if err != nil {
		return err
	}
	e.FieldTwo = int(n)
	e.Parent, err = r.String()
	return err
}

// testService is a minimal in-memory ImportExporter used to exercise the ImEx registry
// without depending on any concrete service. Imports decode the typed payload, allocate
// a fresh key, and persist through the provided transaction; exports look up by key and
// re-encode through Encode.
type testService struct {
	db    *gorp.DB
	table *gorp.Table[string, testEntry]
}

func openTestService(ctx context.Context, db *gorp.DB) *testService {
	GinkgoHelper()
	table := MustSucceed(
		gorp.OpenTable(ctx, gorp.TableConfig[string, testEntry]{DB: db}),
	)
	return &testService{db: db, table: table}
}

func (*testService) Type() resource.Type { return resource.TypeChannel }

func (s *testService) Decode(
	ctx context.Context,
	env imex.Envelope,
) (imex.Write, error) {
	r, err := env.Decode[testResource](ctx)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		opts imex.ImportOptions,
	) (resource.ID, error) {
		key := uuid.New().String()
		e := testEntry{
			Key:      key,
			Name:     env.Name,
			FieldOne: r.FieldOne,
			FieldTwo: r.FieldTwo,
		}
		if !opts.Parent.IsZero() {
			e.Parent = opts.Parent.String()
		}
		if err := s.table.NewCreate().Entry(&e).Exec(ctx, tx); err != nil {
			return resource.ID{}, err
		}
		return resource.ID{Type: resource.TypeChannel, Key: key}, nil
	}, nil
}

func (*testService) Match(map[string]any) bool { return false }

func (s *testService) Export(
	ctx context.Context,
	snap gorp.Snapshot,
	id resource.ID,
) (imex.Envelope, error) {
	var e testEntry
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, testEntry](id.Key)).
		Entry(&e).
		Exec(ctx, snap); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: testVersion,
		Type:    string(resource.TypeChannel),
	}
	if err := env.Encode(
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

func (errorService) Type() resource.Type { return resource.TypeDevice }

func (errorService) Match(map[string]any) bool { return false }

func (errorService) Decode(context.Context, imex.Envelope) (imex.Write, error) {
	return nil, errors.New("importer error: forced failure")
}

func (errorService) Export(
	context.Context,
	gorp.Snapshot,
	resource.ID,
) (imex.Envelope, error) {
	return imex.Envelope{}, errors.New("exporter error: forced failure")
}

type noopImporter struct{ typ resource.Type }

func (noopImporter) Match(map[string]any) bool { return false }

func (n noopImporter) Decode(context.Context, imex.Envelope) (imex.Write, error) {
	return func(context.Context, gorp.Tx, imex.ImportOptions) (resource.ID, error) {
		return resource.ID{Type: n.typ, Key: "noop-key"}, nil
	}, nil
}

// writeErrorImporter decodes every envelope but fails its Write.
type writeErrorImporter struct{ noopImporter }

func (writeErrorImporter) Decode(context.Context, imex.Envelope) (imex.Write, error) {
	return func(context.Context, gorp.Tx, imex.ImportOptions) (resource.ID, error) {
		return resource.ID{}, errors.New("write failed")
	}, nil
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

type noopExporter struct{ typ resource.Type }

func (n noopExporter) Type() resource.Type { return n.typ }

func (n noopExporter) Export(
	context.Context,
	gorp.Snapshot,
	resource.ID,
) (imex.Envelope, error) {
	env := imex.Envelope{Version: testVersion, Type: string(n.typ)}
	if err := env.Encode(testResource{Name: "noop"}); err != nil {
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
		Expect(svc.RegisterImportExporter(ts)).To(Succeed())
		Expect(svc.RegisterImportExporter(errorService{})).To(Succeed())
	})

	Describe("ResolveType", func() {
		typelessEnvelope := func(payload string) imex.Envelope {
			GinkgoHelper()
			var env imex.Envelope
			Expect(json.Unmarshal([]byte(payload), &env)).To(Succeed())
			return env
		}

		It("Should return a non-empty envelope type as-is", func(ctx SpecContext) {
			Expect(svc.ResolveType(imex.Envelope{Type: "anything"})).To(
				Equal("anything"),
			)
		})

		It(
			"Should resolve a typeless envelope through the claiming matcher",
			func(ctx SpecContext) {
				s := imex.NewService()
				Expect(s.RegisterImporter("aaa", matchImporter{
					noopImporter{typ: resource.TypeLog}, "channels",
				})).To(Succeed())
				Expect(s.RegisterImporter("bbb", matchImporter{
					noopImporter{typ: resource.TypeSchematic}, "nodes",
				})).To(Succeed())
				Expect(s.ResolveType(
					typelessEnvelope(`{"version":"1.0.0","nodes":[]}`),
				)).To(Equal("bbb"))
			},
		)

		It(
			"Should offer the body to matchers in sorted type order",
			func(ctx SpecContext) {
				s := imex.NewService()
				Expect(s.RegisterImporter("bbb", matchImporter{
					noopImporter{typ: resource.TypeLog}, "marker",
				})).To(Succeed())
				Expect(s.RegisterImporter("aaa", matchImporter{
					noopImporter{typ: resource.TypeLog}, "marker",
				})).To(Succeed())
				Expect(s.ResolveType(
					typelessEnvelope(`{"marker":true}`),
				)).To(Equal("aaa"))
			},
		)

		It(
			"Should reject a typeless envelope no matcher claims",
			func(ctx SpecContext) {
				Expect(svc.ResolveType(
					typelessEnvelope(`{"version":"1.0.0","foo":1}`),
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
				Expect(s.RegisterImporter("matched", matchImporter{
					noopImporter{typ: resource.TypeLog}, "channels",
				})).To(Succeed())
				id := MustSucceed(s.Import(
					ctx, db, typelessEnvelope(`{"version":"1.0.0","channels":[]}`),
					imex.ImportOptions{FileName: "Legacy.json"},
				))
				Expect(id).To(Equal(resource.ID{
					Type: resource.TypeLog, Key: "noop-key",
				}))
			},
		)
	})

	Describe("RegisterImporter", func() {
		It(
			"Should route each narrow type string to the importer registered under it",
			func(ctx SpecContext) {
				s := imex.NewService()
				Expect(s.RegisterImporter(
					"http_read",
					noopImporter{typ: resource.TypeTask},
				)).To(Succeed())
				Expect(s.RegisterImporter(
					"opc_scan",
					noopImporter{typ: resource.TypeTask},
				)).To(Succeed())
				k1 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "http_read", Name: "ingest"},
					imex.ImportOptions{},
				))
				k2 := MustSucceed(s.Import(
					ctx, db,
					imex.Envelope{Version: 1, Type: "opc_scan", Name: "scan"},
					imex.ImportOptions{},
				))
				Expect(
					k1,
				).To(Equal(resource.ID{Type: resource.TypeTask, Key: "noop-key"}))
				Expect(
					k2,
				).To(Equal(resource.ID{Type: resource.TypeTask, Key: "noop-key"}))
			},
		)
	})

	Describe("RegisterExporter", func() {
		It("Should register an exporter under its own Type", func(ctx SpecContext) {
			s := imex.NewService()
			Expect(
				s.RegisterExporter(noopExporter{typ: resource.TypeLog}),
			).To(Succeed())
			env := MustSucceed(s.Export(ctx, db, resource.ID{
				Type: resource.TypeLog,
				Key:  "any",
			}))
			Expect(env.Type).To(Equal(string(resource.TypeLog)))
			Expect(env.Name).To(Equal("noop"))
		})
	})

	Describe("ExporterRegistered", func() {
		It("Should report whether an exporter is registered for a type", func() {
			s := imex.NewService()
			Expect(
				s.RegisterExporter(noopExporter{typ: resource.TypeLog}),
			).To(Succeed())
			Expect(s.ExporterRegistered(resource.TypeLog)).To(BeTrue())
			Expect(s.ExporterRegistered(resource.TypeRange)).To(BeFalse())
		})
	})

	Describe("Duplicate registration", func() {
		It("Should reject a second importer for a type", func(ctx SpecContext) {
			s := imex.NewService()
			first := noopImporter{typ: resource.TypeChannel}
			Expect(s.RegisterImporter("taken", first)).To(Succeed())
			Expect(s.RegisterImporter(
				"taken",
				noopImporter{typ: resource.TypeLog},
			)).To(SatisfyAll(
				MatchError(imex.ErrTypeTaken),
				MatchError(ContainSubstring(`importer for type "taken"`)),
			))
			Expect(s.Import(
				ctx,
				db,
				imex.Envelope{Version: 1, Type: "taken", Name: "taken"},
				imex.ImportOptions{},
			)).To(HaveField("Type", resource.TypeChannel))
		})

		It("Should reject a second exporter for a type", func(ctx SpecContext) {
			s := imex.NewService()
			Expect(
				s.RegisterExporter(noopExporter{typ: resource.TypeLog}),
			).To(Succeed())
			Expect(s.RegisterExporter(
				noopExporter{typ: resource.TypeLog},
			)).To(SatisfyAll(
				MatchError(imex.ErrTypeTaken),
				MatchError(ContainSubstring(`exporter for type "log"`)),
			))
			env := MustSucceed(s.Export(ctx, db, resource.ID{
				Type: resource.TypeLog,
				Key:  "any",
			}))
			Expect(env.Name).To(Equal("noop"))
		})

		It(
			"Should leave both halves untouched when an import/exporter collides",
			func(ctx SpecContext) {
				s := imex.NewService()
				Expect(s.RegisterImporter(
					string(resource.TypeChannel),
					noopImporter{typ: resource.TypeChannel},
				)).To(Succeed())
				ts := DeferClose(openTestService(ctx, db))
				Expect(s.RegisterImportExporter(ts)).To(MatchError(imex.ErrTypeTaken))
				Expect(s.Export(ctx, db, resource.ID{
					Type: resource.TypeChannel,
					Key:  "any",
				})).Error().To(MatchError(ContainSubstring("no exporter registered")))
			},
		)
	})

	Describe("Import", func() {
		It(
			"Should route to the correct service by type and return the new ID",
			func(ctx SpecContext) {
				id := MustSucceed(svc.Import(
					ctx,
					db,
					sampleEnvelope("Registry Test", resource.TypeChannel),
					imex.ImportOptions{},
				))
				Expect(id.Type).To(Equal(resource.TypeChannel))
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
				svc.Import(ctx, db, env, imex.ImportOptions{}),
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
					ctx, db, sampleEnvelope("Erroring", resource.TypeDevice),
					imex.ImportOptions{},
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
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if _, err := svc.Import(
						ctx,
						tx,
						sampleEnvelope("Good Record", resource.TypeChannel),
						imex.ImportOptions{},
					); err != nil {
						return err
					}
					_, err := svc.Import(ctx, tx, imex.Envelope{
						Version: testVersion,
						Type:    "nonexistent",
						Name:    "Bad Type",
					}, imex.ImportOptions{})
					return err
				})).To(MatchError(ContainSubstring("no importer registered")))
				Expect(ts.Retrieve(ctx, "Good Record")).Error().To(
					MatchError(query.ErrNotFound),
				)
			},
		)
	})

	Describe("Decode", func() {
		It(
			"Should persist nothing until the returned Write runs",
			func(ctx SpecContext) {
				write := MustSucceed(svc.Decode(
					ctx,
					sampleEnvelope("Deferred", resource.TypeChannel),
					"",
				))
				Expect(ts.Retrieve(ctx, "Deferred")).Error().To(
					MatchError(query.ErrNotFound),
				)
				id := MustSucceed(write(ctx, db, imex.ImportOptions{}))
				Expect(MustSucceed(ts.Retrieve(ctx, "Deferred")).Key).
					To(Equal(id.Key))
			},
		)

		It(
			"Should name a nameless envelope after the file name",
			func(ctx SpecContext) {
				write := MustSucceed(svc.Decode(
					ctx, namelessEnvelope(), "dir/Decoded Log.json",
				))
				id := MustSucceed(write(ctx, db, imex.ImportOptions{}))
				Expect(MustSucceed(ts.Retrieve(ctx, "Decoded Log")).Key).
					To(Equal(id.Key))
			},
		)

		It(
			"Should reject an envelope with neither a name nor a file name",
			func(ctx SpecContext) {
				Expect(svc.Decode(ctx, namelessEnvelope(), "")).Error().To(SatisfyAll(
					MatchError(ContainSubstring("name must be a non-empty string")),
					MatchError(validate.ErrValidation),
				))
			},
		)

		It("Should reject an unregistered type", func(ctx SpecContext) {
			Expect(svc.Decode(
				ctx,
				imex.Envelope{Version: testVersion, Type: "nonexistent", Name: "x"},
				"",
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("no importer registered")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should wrap an error from the importer's Decode", func(ctx SpecContext) {
			Expect(svc.Decode(
				ctx, sampleEnvelope("Erroring", resource.TypeDevice), "",
			)).Error().To(MatchError(ContainSubstring(
				"import envelope: importer error: forced failure",
			)))
		})

		It("Should wrap an error from the returned Write", func(ctx SpecContext) {
			s := imex.NewService()
			Expect(s.RegisterImporter("failing", writeErrorImporter{})).To(Succeed())
			write := MustSucceed(s.Decode(
				ctx, imex.Envelope{Version: 1, Type: "failing", Name: "x"}, "",
			))
			Expect(write(ctx, db, imex.ImportOptions{})).Error().To(
				MatchError("import envelope: write failed"),
			)
		})
	})

	Describe("Import Options", func() {
		Describe("FileName", func() {
			It(
				"Should fall back to the extensionless file name when the envelope has no name",
				func(ctx SpecContext) {
					id := MustSucceed(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{
							FileName: "Metrics Log.json",
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
						sampleEnvelope("Body Name", resource.TypeChannel),
						imex.ImportOptions{
							FileName: "File Name.json",
						},
					))
					entry := MustSucceed(ts.Retrieve(ctx, "Body Name"))
					Expect(entry.Key).To(Equal(id.Key))
				},
			)

			DescribeTable("Should strip directory segments from the file name",
				func(ctx SpecContext, fileName, expected string) {
					id := MustSucceed(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{FileName: fileName},
					))
					entry := MustSucceed(ts.Retrieve(ctx, expected))
					Expect(entry.Key).To(Equal(id.Key))
				},
				Entry("slash-separated path", "symbols/Pump.json", "Pump"),
				Entry("backslash-separated path", `symbols\Valve.json`, "Valve"),
				Entry("nested path", "a/b/c/Tank.json", "Tank"),
				Entry("no extension", "symbols/Gauge", "Gauge"),
			)

			It(
				"Should reject an envelope with neither a name nor a file name",
				func(ctx SpecContext) {
					Expect(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{},
					)).Error().To(SatisfyAll(
						MatchError(ContainSubstring("name must be a non-empty string")),
						MatchError(ContainSubstring("validation error")),
					))
				},
			)

			It(
				"Should reject a file name that is only a directory path",
				func(ctx SpecContext) {
					Expect(svc.Import(
						ctx,
						db,
						namelessEnvelope(),
						imex.ImportOptions{
							FileName: "symbols/",
						},
					)).Error().To(
						MatchError(ContainSubstring("name must be a non-empty string")),
					)
				},
			)
		})

		Describe("Parent", func() {
			It("Should import a resource with no parent", func(ctx SpecContext) {
				MustSucceed(svc.Import(
					ctx, db,
					sampleEnvelope("No Parent", resource.TypeChannel),
					imex.ImportOptions{},
				))
				Expect(MustSucceed(ts.Retrieve(ctx, "No Parent")).Parent).To(BeEmpty())
			})

			It("Should pass the parent through to the importer", func(ctx SpecContext) {
				parent := project.ResourceID(uuid.New())
				MustSucceed(svc.Import(
					ctx,
					db,
					sampleEnvelope("Parented", resource.TypeChannel),
					imex.ImportOptions{Parent: parent},
				))
				Expect(MustSucceed(ts.Retrieve(ctx, "Parented")).Parent).
					To(Equal(parent.String()))
			})
		})
	})

	Describe("Export", func() {
		It(
			"Should round-trip a registered resource through Import then Export",
			func(ctx SpecContext) {
				id := MustSucceed(svc.Import(
					ctx, db, sampleEnvelope("Round Trip", resource.TypeChannel),
					imex.ImportOptions{},
				))
				env := MustSucceed(svc.Export(ctx, db, id))
				Expect(env.Version).To(Equal(testVersion))
				Expect(env.Type).To(Equal(string(resource.TypeChannel)))
				Expect(env.Name).To(Equal("Round Trip"))
				roundTripped := MustSucceed(
					WireRoundTrip(env).Decode[testResource](ctx),
				)
				Expect(roundTripped.FieldOne).To(Equal("value"))
				Expect(roundTripped.FieldTwo).To(Equal(42))
			},
		)
		It(
			"Should pass errors from the exporter through verbatim",
			func(ctx SpecContext) {
				Expect(svc.Export(ctx, db, resource.ID{
					Type: resource.TypeDevice,
					Key:  "any-key",
				})).Error().To(MatchError(ContainSubstring("exporter error: forced failure")))
			},
		)

		It("Should reject an unregistered type", func(ctx SpecContext) {
			Expect(svc.Export(ctx, db, resource.ID{
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
			func(ctx SpecContext) {
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
						Expect(s.RegisterImporter(
							t,
							noopImporter{typ: resource.TypeTask},
						)).To(Succeed())
					}(t)
				}
				wg.Wait()
				wg.Add(N)
				for _, t := range types {
					go func(t string) {
						defer wg.Done()
						Expect(s.Import(
							ctx,
							db,
							imex.Envelope{Version: 1, Type: t, Name: t},
							imex.ImportOptions{},
						)).To(HaveField("Type", resource.TypeTask))
					}(t)
				}
				wg.Wait()
			},
		)
	})
})
