package device_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Device", Ordered, func() {
	var (
		db  *gorp.DB
		svc *device.Service
		g   *group.Service
		w   device.Writer
		tx  gorp.Tx
		otg *ontology.Ontology
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g = MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(device.OpenService(ctx, device.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
	})
	AfterAll(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(g.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should correctly create a device", func() {
			d := device.Device{Key: "device1", Rack: 1, Location: "dev1", Name: "Dog"}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should correctly create an ontology resource for the device", func() {
			d := device.Device{Key: "device2", Rack: 1, Location: "dev2", Name: "Cat"}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
		It("Should not recreate the device in the ontology if it already exists", func() {
			d := device.Device{Key: "device3", Rack: 1, Location: "dev3", Name: "Bird"}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(otg.NewWriter(tx).DeleteRelationship(ctx, svc.RootGroup.OntologyID(), ontology.ParentOf, d.OntologyID())).To(Succeed())
			Expect(w.Create(ctx, d)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(svc.RootGroup.OntologyID()).
				TraverseTo(ontology.Children).
				Entry(&res).
				Exec(ctx, tx),
			).To(MatchError(query.NotFound))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a device", func() {
			d := device.Device{Key: "device4", Rack: 1, Location: "dev4", Name: "Fish"}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should retrieve devices by their model", func() {
			d1 := device.Device{Key: "device5", Rack: 1, Location: "dev5", Name: "Fish", Model: "A"}
			d2 := device.Device{Key: "device6", Rack: 1, Location: "dev6", Name: "Fish", Model: "B"}
			d2b := device.Device{Key: "device7", Rack: 1, Location: "dev7", Name: "Fish", Model: "B"}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(svc.NewRetrieve().WhereModels("B").Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(d2, d2b))
		})
		It("Should retrieve devices by their make", func() {
			d1 := device.Device{Key: "device8", Rack: 1, Location: "dev8", Name: "Fish", Make: "A"}
			d2 := device.Device{Key: "device9", Rack: 1, Location: "dev9", Name: "Fish", Make: "B"}
			d2b := device.Device{Key: "device10", Rack: 1, Location: "dev10", Name: "Fish", Make: "B"}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(svc.NewRetrieve().WhereMakes("B").Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(d2, d2b))
		})
		It("Should retrieve devices by their location", func() {
			d1 := device.Device{Key: "device11", Rack: 1, Location: "dev11", Name: "Fish"}
			d2 := device.Device{Key: "device12", Rack: 1, Location: "dev12", Name: "Fish"}
			d2b := device.Device{Key: "device13", Rack: 1, Location: "dev13", Name: "Fish"}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(svc.NewRetrieve().WhereLocations("dev12").Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(d2))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a device", func() {
			d := device.Device{Key: "device14", Rack: 1, Location: "dev14", Name: "Fish"}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
		It("Should correctly delete an ontology resource for the device", func() {
			d := device.Device{Key: "device15", Rack: 1, Location: "dev15", Name: "Fish"}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})
})
