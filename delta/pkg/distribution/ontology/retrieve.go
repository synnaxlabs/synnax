package ontology

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
)

type Retrieve struct {
	txn   gorp.Txn
	exec  func(r Retrieve) error
	query *gorp.Compound[ID, Resource]
}

func newRetrieve(db gorp.Txn, exec func(r Retrieve) error) Retrieve {
	r := Retrieve{
		txn:   db,
		query: &gorp.Compound[ID, Resource]{},
		exec:  exec,
	}
	r.query.Next()
	return r
}

// WhereIDs filters resources by the provided keys.
func (r Retrieve) WhereIDs(keys ...ID) Retrieve {
	r.query.Current().WhereKeys(keys...)
	return r
}

func (r Retrieve) Where(filter func(r *Resource) bool) Retrieve {
	r.query.Current().Where(filter)
	return r
}

type Direction uint8

const (
	Forward  Direction = 1
	Backward Direction = 2
)

type Traverser struct {
	Filter    func(res *Resource, rel *Relationship) bool
	Direction Direction
}

var (
	Children = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == Parent && rel.To == res.ID
		},
		Direction: Backward,
	}
	Parents = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == Parent && rel.From == res.ID
		},
		Direction: Forward,
	}
)

// TraverseTo traverses to the provided relationship type. All filtering methods will
// now be applied to elements of the traversed relationship.
func (r Retrieve) TraverseTo(t Traverser) Retrieve {
	setTraverser(r.query.Current(), t)
	r.query.Next()
	return r
}

// Entry binds the entry that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entry(res *Resource) Retrieve {
	r.query.Current().Entry(res)
	return r
}

// Entries binds a slice that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	r.query.Current().Entries(res)
	return r
}

// WithTxn sets the transaction that the query will use. If not called,
// Retrieve uses the Ontology's database.
func (r Retrieve) WithTxn(txn gorp.Txn) Retrieve { r.txn = txn; return r }

// Exec executes the query.
func (r Retrieve) Exec() error { return r.exec(r) }

const traverseOptKey = "traverse"

func setTraverser(q query.Query, f Traverser) {
	q.Set(traverseOptKey, f)
}

func getTraverser(q query.Query) Traverser {
	return q.GetRequired(traverseOptKey).(Traverser)
}

type retrieve struct {
	services services
}

func (r retrieve) exec(q Retrieve) error {
	var nextIDs []ID
	for i, clause := range q.query.Clauses {
		if i != 0 {
			clause.WhereKeys(nextIDs...)
		}
		if err := clause.Exec(q.txn); err != nil {
			return err
		}
		entries := gorp.GetEntries[ID, Resource](clause)
		resources := entries.All()
		for i, res := range resources {
			data, err := r.services.RetrieveEntity(res.ID)
			if err != nil {
				return err
			}
			res.entity = data
			entries.Set(i, res)
		}
		if len(resources) == 0 {
			break
		}
		if atLast := len(q.query.Clauses) == i+1; atLast {
			return nil
		}
		var err error
		if nextIDs, err = r.traverse(q.txn, getTraverser(clause), resources); err != nil {
			return err
		}
	}
	return nil
}

func (r retrieve) traverse(
	txn gorp.Txn,
	traverse Traverser,
	resources []Resource,
) ([]ID, error) {
	var nextIDs []ID
	return nextIDs, gorp.NewRetrieve[string, Relationship]().
		Where(func(rel *Relationship) bool {
			for _, resource := range resources {
				if traverse.Filter(&resource, rel) {
					if traverse.Direction == Forward {
						nextIDs = append(nextIDs, rel.To)
					} else {
						nextIDs = append(nextIDs, rel.From)
					}
					break
				}
			}
			return false
		}).Exec(txn)
}
