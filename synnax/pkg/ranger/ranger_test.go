package ranger_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/synnax/pkg/ranger"
)

var _ = Describe("Ranger", Ordered, func() {
	var (
		db  *gorp.DB
		svc *ranger.Service
		w   ranger.Writer
		tx  gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.NewService(group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(ranger.NewService(ctx, ranger.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
	})
	AfterAll(func() {
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
		It("Should create a new range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a range by its key", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR).To(Equal(*r))
		})
		It("Should retrieve a range by its name", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereNames(r.Name).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR).To(Equal(*r))
		})
		It("Should retrieve any ranges that overlap a given time range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereOverlapsWith(telem.TimeRange{
				Start: telem.TimeStamp(7 * telem.Second),
				End:   telem.TimeStamp(9 * telem.Second),
			}).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR).To(Equal(*r))
		})
	})
	Describe("Delete", func() {
		It("Should delete a range by its key", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
