package module_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/device/module"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Module", Ordered, func() {
	var (
		db    *gorp.DB
		svc   *module.Service
		w     module.Writer
		tx    gorp.Tx
		rack_ *rack.Rack
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		rackSvc := MustSucceed(rack.OpenService(ctx, rack.Config{DB: db, Ontology: otg, Group: g, Host: mock.StaticHostKeyProvider(1)}))
		svc = MustSucceed(module.OpenService(ctx, module.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
		}))
		rack_ = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(db).Create(ctx, rack_)).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
		Expect(svc.Close()).To(Succeed())
	})
	Describe("Key", func() {
		It("Should construct and deconstruct a key from its components", func() {
			rk := rack.NewKey(core.NodeKey(1), 2)
			k := module.NewKey(rk, 2)
			Expect(k.Rack()).To(Equal(rk))
			Expect(k.LocalKey()).To(Equal(uint32(2)))
		})
	})
	Describe("Create", func() {
		It("Should correctly create a module and assign it a unique key", func() {
			m := &module.Module{
				Key:  module.NewKey(rack_.Key, 0),
				Name: "Test Module",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(module.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Module"))
		})
		It("Should correctly increment the module count", func() {
			m := &module.Module{
				Key:  module.NewKey(rack_.Key, 0),
				Name: "Test Module",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(module.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Module"))
			m = &module.Module{
				Key:  module.NewKey(rack_.Key, 0),
				Name: "Test Module",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(module.NewKey(rack_.Key, 2)))
			Expect(m.Name).To(Equal("Test Module"))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a module", func() {
			m := &module.Module{
				Key:  module.NewKey(rack_.Key, 0),
				Name: "Test Module",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(module.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Module"))
			var res module.Module
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*m))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a module", func() {
			m := &module.Module{
				Key:  module.NewKey(rack_.Key, 0),
				Name: "Test Module",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(module.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Module"))
			Expect(w.Delete(ctx, m.Key)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})
})
