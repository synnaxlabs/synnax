// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"iter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

const fakeConfigType ontology.ResourceType = "task_config_fake"

// fakeConfigOntologyService resolves the fake config resources the fake provider
// defines, so ontology traversals over them succeed in tests.
type fakeConfigOntologyService struct {
	observe.Noop[iter.Seq[ontology.Change]]
}

var _ ontology.Service = (*fakeConfigOntologyService)(nil)

func (f *fakeConfigOntologyService) Type() ontology.ResourceType { return fakeConfigType }

func (f *fakeConfigOntologyService) Schema() zyn.Schema { return zyn.Object(nil) }

func (f *fakeConfigOntologyService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.Resource{
		ID:   ontology.ID{Type: fakeConfigType, Key: key},
		Name: key,
	}, nil
}

type fakeProvider struct {
	otg           *ontology.Ontology
	types         []string
	configs       map[task.Key]msgpack.EncodedJSON
	createCalls   int
	copyCalls     int
	deleteCalls   int
	lastCreateCfg msgpack.EncodedJSON
}

var _ task.ConfigProvider = (*fakeProvider)(nil)

func newFakeProvider(otg *ontology.Ontology, types ...string) *fakeProvider {
	return &fakeProvider{
		otg:     otg,
		types:   types,
		configs: make(map[task.Key]msgpack.EncodedJSON),
	}
}

func fakeConfigID(key task.Key) ontology.ID {
	return ontology.ID{Type: fakeConfigType, Key: key.String()}
}

func (f *fakeProvider) Types() []string { return f.types }

func (f *fakeProvider) Create(
	ctx context.Context,
	tx gorp.Tx,
	key task.Key,
	_ string,
	cfg msgpack.EncodedJSON,
) (ontology.ID, error) {
	f.createCalls++
	f.lastCreateCfg = cfg
	f.configs[key] = cfg
	id := fakeConfigID(key)
	return id, f.otg.NewWriter(tx).DefineResource(ctx, id)
}

func (f *fakeProvider) Load(
	_ context.Context,
	_ gorp.Tx,
	key task.Key,
) (msgpack.EncodedJSON, error) {
	cfg, ok := f.configs[key]
	if !ok {
		return nil, errors.Wrapf(query.ErrNotFound, "no config for task %v", key)
	}
	return cfg, nil
}

func (f *fakeProvider) Copy(
	ctx context.Context,
	tx gorp.Tx,
	from, to task.Key,
) (ontology.ID, error) {
	f.copyCalls++
	cfg, ok := f.configs[from]
	if !ok {
		return ontology.ID{}, errors.Wrapf(query.ErrNotFound, "no config for task %v", from)
	}
	f.configs[to] = cfg
	id := fakeConfigID(to)
	return id, f.otg.NewWriter(tx).DefineResource(ctx, id)
}

func (f *fakeProvider) Delete(ctx context.Context, tx gorp.Tx, key task.Key) error {
	f.deleteCalls++
	delete(f.configs, key)
	return f.otg.NewWriter(tx).DeleteResource(ctx, fakeConfigID(key))
}

var _ = Describe("ConfigProvider", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *task.Service
		otg      *ontology.Ontology
		w        task.Writer
		tx       gorp.Tx
		testRack *rack.Rack
		provider *fakeProvider
	)
	BeforeAll(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackService := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.StaticHostKeyProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackService,
			Status:   stat,
			Search:   searchIdx,
		}))
		otg.RegisterService(&fakeConfigOntologyService{})
		provider = newFakeProvider(otg, "fake_typed", "fake_typed_2")
		Expect(svc.RegisterConfigProvider(provider)).To(Succeed())
		testRack = &rack.Rack{Name: "Config Test Rack"}
		Expect(rackService.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	})
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func(ctx SpecContext) {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("RegisterConfigProvider", func() {
		It("Should reject a provider for an already-claimed task type", func(ctx SpecContext) {
			other := newFakeProvider(otg, "fake_typed")
			Expect(svc.RegisterConfigProvider(other)).To(
				MatchError(ContainSubstring("already registered")),
			)
		})
	})

	Describe("Create", func() {
		It("Should route the config to the provider and link it to the task", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "value"}
			t := task.Task{Key: task.NewKey(testRack.Key, 0), Type: "fake_typed", Name: "Typed", Config: cfg}
			createCalls := provider.createCalls
			Expect(w.Create(ctx, &t)).To(Succeed())
			Expect(provider.createCalls).To(Equal(createCalls + 1))
			Expect(provider.lastCreateCfg).To(Equal(cfg))
			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(task.OntologyID(t.Key)).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID).To(Equal(fakeConfigID(t.Key)))
		})

		It("Should keep storing the config on the task while dual-writing", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "stored"}
			t := task.Task{Key: task.NewKey(testRack.Key, 0), Type: "fake_typed", Name: "Stored", Config: cfg}
			Expect(w.Create(ctx, &t)).To(Succeed())
			var stored task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Entry(&stored).
				Exec(ctx, tx)).To(Succeed())
			Expect(stored.Config).To(Equal(cfg))
		})

		It("Should not dispatch internal tasks to the provider", func(ctx SpecContext) {
			createCalls := provider.createCalls
			t := task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Type:     "fake_typed",
				Name:     "Internal",
				Internal: true,
				Config:   msgpack.EncodedJSON{"param": "internal"},
			}
			Expect(w.Create(ctx, &t)).To(Succeed())
			Expect(provider.createCalls).To(Equal(createCalls))
		})

		It("Should leave task types without a provider on the fallback path", func(ctx SpecContext) {
			createCalls := provider.createCalls
			cfg := msgpack.EncodedJSON{"param": "fallback"}
			t := task.Task{Key: task.NewKey(testRack.Key, 0), Type: "untyped", Name: "Fallback", Config: cfg}
			Expect(w.Create(ctx, &t)).To(Succeed())
			Expect(provider.createCalls).To(Equal(createCalls))
			var stored task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Entry(&stored).
				Exec(ctx, tx)).To(Succeed())
			Expect(stored.Config).To(Equal(cfg))
		})

		It("Should hand the preserved config to the provider when updating a snapshot", func(ctx SpecContext) {
			original := msgpack.EncodedJSON{"param": "original"}
			t := task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Type:     "fake_typed",
				Name:     "Snapshot",
				Snapshot: true,
				Config:   original,
			}
			Expect(w.Create(ctx, &t)).To(Succeed())
			updated := t
			updated.Config = msgpack.EncodedJSON{"param": "updated"}
			Expect(w.Create(ctx, &updated)).To(Succeed())
			Expect(provider.lastCreateCfg).To(Equal(original))
		})
	})

	Describe("ResolveConfigs", func() {
		It("Should fill configs from the provider", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "typed"}
			t := task.Task{Key: task.NewKey(testRack.Key, 0), Type: "fake_typed", Name: "Resolve", Config: cfg}
			Expect(w.Create(ctx, &t)).To(Succeed())
			resolved := msgpack.EncodedJSON{"param": "resolved"}
			provider.configs[t.Key] = resolved
			tasks := []task.Task{{Key: t.Key, Type: t.Type}}
			Expect(svc.ResolveConfigs(ctx, tx, tasks)).To(Succeed())
			Expect(tasks[0].Config).To(Equal(resolved))
		})

		It("Should keep the stored config when the provider has no record", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "blob"}
			tasks := []task.Task{{
				Key:    task.NewKey(testRack.Key, 40),
				Type:   "fake_typed",
				Config: cfg,
			}}
			Expect(svc.ResolveConfigs(ctx, tx, tasks)).To(Succeed())
			Expect(tasks[0].Config).To(Equal(cfg))
		})

		It("Should leave tasks without a provider untouched", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "untyped"}
			tasks := []task.Task{{
				Key:    task.NewKey(testRack.Key, 41),
				Type:   "untyped",
				Config: cfg,
			}}
			Expect(svc.ResolveConfigs(ctx, tx, tasks)).To(Succeed())
			Expect(tasks[0].Config).To(Equal(cfg))
		})

		It("Should leave internal tasks untouched", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "internal"}
			tasks := []task.Task{{
				Key:      task.NewKey(testRack.Key, 42),
				Type:     "fake_typed",
				Internal: true,
				Config:   cfg,
			}}
			Expect(svc.ResolveConfigs(ctx, tx, tasks)).To(Succeed())
			Expect(tasks[0].Config).To(Equal(cfg))
		})
	})

	Describe("Delete", func() {
		It("Should delete the provider's config record with the task", func(ctx SpecContext) {
			t := task.Task{
				Key:    task.NewKey(testRack.Key, 0),
				Type:   "fake_typed",
				Name:   "Delete",
				Config: msgpack.EncodedJSON{"param": "delete"},
			}
			Expect(w.Create(ctx, &t)).To(Succeed())
			deleteCalls := provider.deleteCalls
			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Expect(provider.deleteCalls).To(Equal(deleteCalls + 1))
			Expect(provider.configs).ToNot(HaveKey(t.Key))
		})

		It("Should not dispatch internal task deletion to the provider", func(ctx SpecContext) {
			t := task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Type:     "fake_typed",
				Name:     "Internal Delete",
				Internal: true,
			}
			Expect(w.Create(ctx, &t)).To(Succeed())
			deleteCalls := provider.deleteCalls
			Expect(w.Delete(ctx, t.Key, true)).To(Succeed())
			Expect(provider.deleteCalls).To(Equal(deleteCalls))
		})

		It("Should delete a task that does not exist without provider calls", func(ctx SpecContext) {
			deleteCalls := provider.deleteCalls
			Expect(w.Delete(ctx, task.NewKey(testRack.Key, 60000), false)).To(Succeed())
			Expect(provider.deleteCalls).To(Equal(deleteCalls))
		})
	})

	Describe("Copy", func() {
		It("Should copy the provider's config record and link the new task", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "copy"}
			t := task.Task{Key: task.NewKey(testRack.Key, 0), Type: "fake_typed", Name: "Source", Config: cfg}
			Expect(w.Create(ctx, &t)).To(Succeed())
			copied := MustSucceed(w.Copy(ctx, t.Key, "Copy", false))
			Expect(provider.configs[copied.Key]).To(Equal(cfg))
			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(task.OntologyID(copied.Key)).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID).To(Equal(fakeConfigID(copied.Key)))
		})
	})

	Describe("DecodeConfig", func() {
		type decoded struct{ Value string }
		failing := func(msgpack.EncodedJSON) (decoded, error) {
			return decoded{}, errors.New("current schema mismatch")
		}
		succeeding := func(cfg msgpack.EncodedJSON) (decoded, error) {
			return decoded{Value: cfg["param"].(string)}, nil
		}

		It("Should return the first successful decode", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "v"}
			Expect(task.DecodeConfig(cfg, succeeding, failing)).To(Equal(decoded{Value: "v"}))
		})

		It("Should fall back to older decoders", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"param": "old"}
			Expect(task.DecodeConfig(cfg, failing, succeeding)).To(Equal(decoded{Value: "old"}))
		})

		It("Should return the first decoder's error when all fail", func(ctx SpecContext) {
			Expect(task.DecodeConfig(msgpack.EncodedJSON{}, failing, failing)).Error().
				To(MatchError(ContainSubstring("current schema mismatch")))
		})

		It("Should error when no decoders are provided", func(ctx SpecContext) {
			Expect(task.DecodeConfig[decoded](msgpack.EncodedJSON{})).Error().
				To(MatchError(task.ErrNoConfigDecoders))
		})
	})
})
