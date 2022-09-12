package rbac

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/arya-analytics/x/gorp"
)

type Legislator struct {
	DB *gorp.DB
}

func (l *Legislator) Create(txn gorp.Txn, p Policy) error {
	return gorp.NewCreate[string, Policy]().Entry(&p).Exec(txn)
}

func (l *Legislator) Retrieve(subject, object ontology.ID) ([]Policy, error) {
	var p []Policy
	return p, gorp.NewRetrieve[string, Policy]().
		WhereKeys(NewPolicyKey(subject, object)).
		Entries(&p).
		Exec(l.DB)
}
