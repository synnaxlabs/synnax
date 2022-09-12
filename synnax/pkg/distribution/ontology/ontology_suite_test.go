package ontology_test

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type emptyService struct{}

const emptyType ontology.Type = "empty"

func newEmptyID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: emptyType}
}

func (s *emptyService) Schema() *ontology.Schema {
	return &ontology.Schema{
		Type: emptyType,
		Fields: map[string]schema.Field{
			"key": {Type: schema.String},
		},
	}
}

func (s *emptyService) RetrieveEntity(key string) (ontology.Entity, error) {
	e := schema.NewEntity(s.Schema())
	schema.Set(e, "key", key)
	return e, nil
}

var (
	db  *gorp.DB
	otg *ontology.Ontology
	txn gorp.Txn
)

var _ = BeforeSuite(func() {
	var err error
	db = gorp.Wrap(memkv.New())
	otg, err = ontology.Open(gorp.Wrap(db))
	Expect(err).ToNot(HaveOccurred())
	otg.RegisterService(&emptyService{})
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	txn = db.BeginTxn()
})

var _ = AfterEach(func() {
	Expect(txn.Close()).To(Succeed())
})

func TestOntology(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ontology Suite")
}
