package ontology

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

type (
	Schema = schema.Schema
	Entity = schema.Entity
	Type   = schema.Type
)

type Ontology struct {
	db       *gorp.DB
	retrieve retrieve
}

// Open opens the ontology stored in the given database.
func Open(db *gorp.DB) (*Ontology, error) {
	o := &Ontology{
		db:       db,
		retrieve: retrieve{services: make(services)},
	}
	err := o.NewRetrieve().WhereIDs(Root).Exec()
	if errors.Is(err, query.NotFound) {
		if err := o.NewWriter().DefineResource(Root); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

type Writer interface {
	// DefineResource defines a new resource with the given ID. If the resource already
	// exists, DefineResource does nothing.
	DefineResource(id ID) error
	// DeleteResource deletes the resource with the given ID along with all of its
	// incoming and outgoing relationships.  If the resource does not exist,
	// DeleteResource does nothing.
	DeleteResource(id ID) error
	// DefineRelationship defines a directional relationship of type t between the
	// resources with the given IDs. If the relationship already exists, DefineRelationship
	// does nothing.
	DefineRelationship(from, to ID, t RelationshipType) error
	// DeleteRelationship deletes the relationship with the given IDs and type. If the
	// relationship does not exist, DeleteRelationship does nothing.
	DeleteRelationship(from, to ID, t RelationshipType) error
	// NewRetrieve opens a new Retrieve query that uses the Writer's transaction.
	NewRetrieve() Retrieve
}

// NewRetrieve opens a new Retrieve query, which is used to traverse the ontology.
func (o *Ontology) NewRetrieve() Retrieve { return newRetrieve(o.db, o.retrieve.exec) }

// NewWriter opens a new Writer.
func (o *Ontology) NewWriter() Writer { return o.NewWriterUsingTxn(o.db) }

// NewWriterUsingTxn opens a new Writer using the provided transaction.
// Panics if the transaction does not root from the same database as the Ontology.
func (o *Ontology) NewWriterUsingTxn(txn gorp.Txn) Writer {
	return dagWriter{txn: txn, retrieve: o.retrieve}
}

// RegisterService registers a Service for a particular Type with the Ontology.
// Ontology will execute queries for Entity information for the given Type using the
// provided Service. RegisterService panics if a Service is already registered for
// the given Type.
func (o *Ontology) RegisterService(s Service) { o.retrieve.services.Register(s) }
