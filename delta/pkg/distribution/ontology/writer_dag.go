package ontology

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

// dagWriter is a key-value backed directed acyclic graph that implements the Writer
// interface.
type dagWriter struct {
	txn      gorp.Txn
	retrieve retrieve
}

var CyclicDependency = errors.New("[ontology] cyclic dependency")

// DefineResource implements the Writer interface.
func (d dagWriter) DefineResource(tk ID) error {
	if err := tk.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[ID, Resource]().
		Entry(&Resource{ID: tk}).
		Exec(d.txn)
}

// DeleteResource implements the Writer interface.
func (d dagWriter) DeleteResource(tk ID) error {
	if err := d.deleteIncomingRelationships(tk); err != nil {
		return err
	}
	if err := d.deleteOutgoingRelationships(tk); err != nil {
		return err
	}
	return gorp.NewDelete[ID, Resource]().WhereKeys(tk).Exec(d.txn)
}

// DefineRelationship implements the Writer interface.
func (d dagWriter) DefineRelationship(from, to ID, t RelationshipType) error {
	rel := Relationship{From: from, To: to, Type: t}
	exists, err := d.checkRelationshipExists(rel)
	if err != nil || exists {
		return err
	}
	if err := d.validateResourcesExist(from, to); err != nil {
		return err
	}
	descendants, err := d.retrieveDescendants(to)
	if err != nil {
		return err
	}
	if _, exists := descendants[from]; exists {
		return CyclicDependency
	}
	return gorp.NewCreate[string, Relationship]().Entry(&rel).Exec(d.txn)

}

// DeleteRelationship implements the Writer interface.
func (d dagWriter) DeleteRelationship(from, to ID, t RelationshipType) error {
	return gorp.NewDelete[string, Relationship]().
		WhereKeys(Relationship{From: from, To: to, Type: t}.GorpKey()).
		Exec(d.txn)
}

// NewRetrieve implements the Writer interface.
func (d dagWriter) NewRetrieve() Retrieve { return newRetrieve(d.txn, d.retrieve.exec) }

func (d dagWriter) retrieveOutgoingRelationships(key ID) ([]Resource, error) {
	relationships, err := d.retrieveRelationships(func(rel *Relationship) bool {
		return rel.From == key
	})
	if err != nil {
		return nil, err
	}
	var keys []ID
	for _, rel := range relationships {
		keys = append(keys, rel.To)
	}
	return d.retrieveResources(keys)
}

func (d dagWriter) retrieveRelationships(matcher func(*Relationship) bool) ([]Relationship, error) {
	var relationships []Relationship
	return relationships, gorp.NewRetrieve[string, Relationship]().
		Where(matcher).
		Entries(&relationships).
		Exec(d.txn)
}

func (d dagWriter) retrieveResources(keys []ID) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[ID, Resource]().
		WhereKeys(keys...).
		Entries(&resources).
		Exec(d.txn)
}

func (d dagWriter) retrieveDescendants(key ID) (map[ID]Resource, error) {
	descendants := make(map[ID]Resource)
	children, err := d.retrieveOutgoingRelationships(key)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	for _, child := range children {
		childDescendants, err := d.retrieveDescendants(child.ID)
		if err != nil {
			return nil, err
		}
		for k, v := range childDescendants {
			descendants[k] = v
		}
		descendants[child.ID] = child
	}
	return descendants, nil
}

func (d dagWriter) deleteIncomingRelationships(tk ID) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel *Relationship) bool {
		return rel.To == tk
	}).Exec(d.txn)
}

func (d dagWriter) deleteOutgoingRelationships(tk ID) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel *Relationship) bool {
		return rel.From == tk
	}).Exec(d.txn)
}

func (d dagWriter) checkRelationshipExists(rel Relationship) (bool, error) {
	exists, err := gorp.NewRetrieve[string, Relationship]().
		WhereKeys(rel.GorpKey()).
		Exists(d.txn)
	if err != nil {
		return false, err
	}
	reverseRel := Relationship{From: rel.To, To: rel.From, Type: rel.Type}
	reverseExists, err := gorp.NewRetrieve[string, Relationship]().
		WhereKeys(reverseRel.GorpKey()).
		Exists(d.txn)
	if err != nil {
		return false, err
	}
	if reverseExists {
		return true, CyclicDependency
	}
	return exists, nil
}

func (d dagWriter) validateResourcesExist(ids ...ID) error {
	ok, err := gorp.NewRetrieve[ID, Resource]().WhereKeys(ids...).Exists(d.txn)
	if err != nil {
		return err
	}
	if !ok {
		return errors.WithDetailf(
			query.NotFound,
			"[ontology] - resources %v not found",
			ids,
		)
	}
	return nil
}
