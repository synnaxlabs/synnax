package rack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("Rack", Ordered, func() {
	var (
		db  *gorp.DB
		svc *rack.Service
		w   rack.Writer
		tx  gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Host:     mock.StaticHostKeyProvider(1),
		}))
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Key", func() {
		It("Should correctly construct and deconstruct key from its components", func() {
			k := rack.NewKey(1, 2)
			Expect(k.Node()).To(Equal(core.NodeKey(1)))
			Expect(k.LocalKey()).To(Equal(uint16(2)))
		})
	})
	Describe("Create", func() {
		It("Should create a rack and assign it a key", func() {
			r := &rack.Rack{Name: "rack1"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key.IsValid()).To(BeTrue())
			Expect(r.Key.Node()).To(Equal(core.NodeKey(1)))
			Expect(r.Key.LocalKey()).To(Equal(uint16(1)))
		})
		It("Should correctly increment the local key counter", func() {
			r := &rack.Rack{Name: "rack2"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key.LocalKey()).To(Equal(uint16(2)))
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a rack by its key", func() {
			r := &rack.Rack{Name: "rack3"}
			Expect(w.Create(ctx, r)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*r))
		})
	})
	Describe("Delete", func() {
		It("Should delete a rack by its key", func() {
			r := &rack.Rack{Name: "rack4"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(w.Delete(ctx, r.Key)).To(Succeed())
			var res rack.Rack
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&res).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})
})
