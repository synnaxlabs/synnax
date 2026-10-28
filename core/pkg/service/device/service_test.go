// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"
	"iter"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

const resourceType = ontology.ResourceTypeDevice

type module struct {
	Key     uuid.UUID `json:"key"     msgpack:"key"`
	Rack    rack.Key  `json:"rack"    msgpack:"rack"`
	Name    string    `json:"name"    msgpack:"name"`
	Chassis uuid.UUID `json:"chassis" msgpack:"chassis"`
}

func (m module) GorpKey() uuid.UUID { return m.Key }

func (m module) SetOptions() []any { return nil }

func (m *module) SetKey(key uuid.UUID) { m.Key = key }

func (m module) EncodeOrc(w *orc.Writer) error {
	w.Write(m.Key[:])
	w.Uint32(uint32(m.Rack))
	w.String(m.Name)
	w.Write(m.Chassis[:])
	return nil
}

func (m *module) DecodeOrc(r *orc.Reader) error {
	if _, err := r.Read(m.Key[:]); err != nil {
		return err
	}
	rawRack, err := r.Uint32()
	if err != nil {
		return err
	}
	m.Rack = rack.Key(rawRack)
	if m.Name, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(m.Chassis[:])
	return err
}

func (m *module) Validate() error {
	v := validate.New("module")
	v.NotEmptyString("name", m.Name)
	return v.Error()
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"rack": zyn.Uint32().Coerce(),
	"name": zyn.String(),
})

var _ = Describe("Service", func() {
	var (
		otg        *ontology.Ontology
		stat       *status.Service
		rackSvc    *rack.Service
		svc        *device.Service[module]
		referenced set.Set[uuid.UUID]
		deleted    []uuid.UUID
		tx         gorp.Tx
	)
	newConfig := func() device.ServiceConfig[module] {
		search := gorp.NewSearchIndex[uuid.UUID, module](
			"search",
			func(m *module) []string { return []string{m.Name} },
		)
		return device.ServiceConfig[module]{
			DB:          db,
			Ontology:    otg,
			Indexes:     []gorp.Index[uuid.UUID, module]{search},
			Search:      search,
			Status:      stat,
			Type:        resourceType,
			Schema:      schema,
			SetEntryKey: (*module).SetKey,
			NameOf:      func(m module) string { return m.Name },
			RackOf:      func(m module) rack.Key { return m.Rack },
		}
	}
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
		}))
		referenced = set.New[uuid.UUID]()
		deleted = nil
		cfg := newConfig()
		cfg.ValidateEntry = (*module).Validate
		cfg.ParentOf = func(m module) (ontology.ID, bool) {
			return ontology.ID{Type: resourceType, Key: m.Chassis.String()},
				m.Chassis != uuid.Nil()
		}
		cfg.StatusDetails = func(m module) any {
			return map[string]any{"rack": m.Rack}
		}
		cfg.Referenced = func(
			_ context.Context,
			_ gorp.Tx,
			key uuid.UUID,
		) (bool, error) {
			return referenced.Contains(key), nil
		}
		cfg.DeleteChildren = func(_ context.Context, _ gorp.Tx, key uuid.UUID) error {
			deleted = append(deleted, key)
			return nil
		}
		svc = MustOpen(device.OpenService(ctx, cfg))
		tx = DeferClose(db.OpenTx())
	})

	retrieve := func(ctx context.Context, key uuid.UUID) module {
		GinkgoHelper()
		var m module
		Expect(svc.NewRetrieve().
			Where(gorp.MatchKeys[uuid.UUID, module](key)).
			Entry(&m).
			Exec(ctx, tx)).To(Succeed())
		return m
	}
	retrieveStatus := func(ctx context.Context, key uuid.UUID) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.MatchKeys[status.Key, status.Status[any]](
				svc.OntologyID(key).String(),
			)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	parents := func(ctx context.Context, key uuid.UUID) []ontology.ID {
		GinkgoHelper()
		var res []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(svc.OntologyID(key)).
			TraverseTo(ontology.ParentsTraverser).
			Entries(&res).
			Exec(ctx, tx)).To(Succeed())
		return ontology.ResourceIDs(res)
	}

	Describe("OpenService", func() {
		It("Should reject a config missing the name hook", func(ctx SpecContext) {
			cfg := newConfig()
			cfg.NameOf = nil
			Expect(device.OpenService(ctx, cfg)).Error().
				To(MatchError(ContainSubstring("name_of: must be non-nil")))
		})
		It("Should reject a config missing the search index", func(ctx SpecContext) {
			cfg := newConfig()
			cfg.Search = nil
			Expect(device.OpenService(ctx, cfg)).Error().
				To(MatchError(ContainSubstring("search: must be non-nil")))
		})
		It("Should reject a config missing the type", func(ctx SpecContext) {
			cfg := newConfig()
			cfg.Type = ""
			Expect(device.OpenService(ctx, cfg)).Error().
				To(MatchError(ContainSubstring("type: required")))
		})
	})

	Describe("Create", func() {
		It("Should mint a key and store the resource", func(ctx SpecContext) {
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, m.Key)).To(Equal(m))
		})
		It("Should keep a caller-provided key", func(ctx SpecContext) {
			key := uuid.New()
			m := module{Key: key, Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			Expect(m.Key).To(Equal(key))
		})
		It("Should return the resource's validation error", func(ctx SpecContext) {
			m := module{Rack: rackSvc.EmbeddedKey}
			Expect(svc.Create(ctx, tx, &m)).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should define the ontology resource under the rack", func(ctx SpecContext) {
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(svc.OntologyID(m.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("Module"))
			Expect(parents(ctx, m.Key)).To(ConsistOf(rackSvc.EmbeddedKey.OntologyID()))
		})
		It(
			"Should define the ontology resource under its parent",
			func(ctx SpecContext) {
				chassis := module{Rack: rackSvc.EmbeddedKey, Name: "Chassis"}
				Expect(svc.Create(ctx, tx, &chassis)).To(Succeed())
				m := module{
					Rack:    rackSvc.EmbeddedKey,
					Name:    "Module",
					Chassis: chassis.Key,
				}
				Expect(svc.Create(ctx, tx, &m)).To(Succeed())
				Expect(parents(ctx, m.Key)).To(ConsistOf(svc.OntologyID(chassis.Key)))
			},
		)
		It("Should write an unknown status with the resource's details", func(
			ctx SpecContext,
		) {
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			s := retrieveStatus(ctx, m.Key)
			Expect(s.Name).To(Equal("Module"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(s.Details).To(HaveKeyWithValue("rack", BeNumerically("==", m.Rack)))
		})
		It("Should keep an existing status on a second create", func(ctx SpecContext) {
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			healthy := status.Status[any]{
				Key:     svc.OntologyID(m.Key).String(),
				Name:    m.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "connected",
			}
			Expect(stat.NewWriter(tx).Set(ctx, &healthy)).To(Succeed())
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			Expect(retrieveStatus(ctx, m.Key).Variant).To(Equal(status.VariantSuccess))
		})
	})

	Describe("Update", func() {
		var m module
		BeforeEach(func(ctx SpecContext) {
			m = module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
		})
		It("Should store the changed resource", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				m.Name = "Renamed"
				return m, nil
			})).To(Succeed())
			Expect(retrieve(ctx, m.Key).Name).To(Equal("Renamed"))
		})
		It("Should rename the status with the resource", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				m.Name = "Renamed"
				return m, nil
			})).To(Succeed())
			Expect(retrieveStatus(ctx, m.Key).Name).To(Equal("Renamed"))
		})
		It("Should move the parent edge when the rack changes", func(ctx SpecContext) {
			r := rack.Rack{Name: "Rack 2"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				m.Rack = r.Key
				return m, nil
			})).To(Succeed())
			Expect(parents(ctx, m.Key)).To(ConsistOf(r.Key.OntologyID()))
		})
		It("Should keep the stored key", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				m.Key = uuid.New()
				return m, nil
			})).To(Succeed())
			Expect(retrieve(ctx, m.Key).Key).To(Equal(m.Key))
		})
		It("Should return the resource's validation error", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				m.Name = ""
				return m, nil
			})).To(MatchError(ContainSubstring("name: required")))
			Expect(retrieve(ctx, m.Key).Name).To(Equal("Module"))
		})
		It("Should return the change's error", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, m.Key, func(m module) (module, error) {
				return m, errors.New("nope")
			})).To(MatchError("nope"))
		})
		It("Should keep the stored resource when a write without a tx fails", func(
			ctx SpecContext,
		) {
			stored := module{Rack: rackSvc.EmbeddedKey, Name: "Stored"}
			Expect(svc.Create(ctx, nil, &stored)).To(Succeed())
			Expect(svc.Update(ctx, nil, stored.Key, func(m module) (module, error) {
				m.Chassis = uuid.New()
				return m, nil
			})).To(MatchError(query.ErrNotFound))
			Expect(retrieve(ctx, stored.Key)).To(Equal(stored))
			Expect(parents(ctx, stored.Key)).
				To(ConsistOf(rackSvc.EmbeddedKey.OntologyID()))
		})
		It("Should return not found for a missing resource", func(ctx SpecContext) {
			Expect(svc.Update(ctx, tx, uuid.New(), func(m module) (module, error) {
				return m, nil
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		var m module
		BeforeEach(func(ctx SpecContext) {
			m = module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
		})
		It("Should remove the resource, its status, and its ontology resource", func(
			ctx SpecContext,
		) {
			Expect(svc.Delete(ctx, tx, m.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(gorp.MatchKeys[uuid.UUID, module](m.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
				Where(gorp.MatchKeys[status.Key, status.Status[any]](
					svc.OntologyID(m.Key).String(),
				)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(otg.NewRetrieve().
				WhereIDs(svc.OntologyID(m.Key)).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
		It("Should refuse while a task references the resource", func(ctx SpecContext) {
			referenced.Add(m.Key)
			Expect(svc.Delete(ctx, tx, m.Key)).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, m.Key)).To(Equal(m))
		})
		It("Should delete the resource's children first", func(ctx SpecContext) {
			Expect(svc.Delete(ctx, tx, m.Key)).To(Succeed())
			Expect(deleted).To(Equal([]uuid.UUID{m.Key}))
		})
		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.Delete(ctx, tx, m.Key)).To(Succeed())
			Expect(svc.Delete(ctx, tx, m.Key)).To(Succeed())
		})
		It("Should leave every resource when one is referenced", func(ctx SpecContext) {
			other := module{Rack: rackSvc.EmbeddedKey, Name: "Other"}
			Expect(svc.Create(ctx, tx, &other)).To(Succeed())
			referenced.Add(other.Key)
			Expect(svc.Delete(ctx, tx, m.Key, other.Key)).
				To(MatchError(validate.ErrValidation))
			Expect(deleted).To(BeEmpty())
			Expect(retrieveStatus(ctx, m.Key).Name).To(Equal("Module"))
			Expect(parents(ctx, m.Key)).To(ConsistOf(rackSvc.EmbeddedKey.OntologyID()))
		})
	})

	Describe("OnChange", func() {
		It("Should translate committed writes into ontology changes", func(
			ctx SpecContext,
		) {
			var changes []ontology.Change
			disconnect := svc.OnChange(
				func(_ context.Context, seq iter.Seq[ontology.Change]) {
					for c := range seq {
						changes = append(changes, c)
					}
				},
			)
			defer disconnect()
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Variant).To(Equal(change.VariantSet))
			Expect(changes[0].Key).To(Equal(svc.OntologyID(m.Key).String()))
			Expect(changes[0].Value.Name).To(Equal("Module"))
			deleteTx := DeferClose(db.OpenTx())
			Expect(svc.Delete(ctx, deleteTx, m.Key)).To(Succeed())
			Expect(deleteTx.Commit(ctx)).To(Succeed())
			Expect(changes).To(HaveLen(2))
			Expect(changes[1].Variant).To(Equal(change.VariantDelete))
			Expect(changes[1].Key).To(Equal(svc.OntologyID(m.Key).String()))
			Expect(changes[1].Value).To(BeZero())
		})
	})

	Describe("Search", func() {
		It("Should find a resource by name", func(ctx SpecContext) {
			m := module{Rack: rackSvc.EmbeddedKey, Name: "Thermocouple module"}
			Expect(svc.Create(ctx, tx, &m)).To(Succeed())
			res := MustSucceed(svc.Search(ctx, tx, "thermocouple", 10))
			Expect(res.Keys).To(ConsistOf(svc.OntologyID(m.Key)))
		})
	})
})
