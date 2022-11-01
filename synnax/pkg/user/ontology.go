package user

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

const ontologyType ontology.Type = "user"

func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: key.String()}
}

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":      {Type: schema.UUID},
		"username": {Type: schema.String},
	},
}

var _ ontology.Service = (*Service)(nil)

// Schema implements the ontology.Service interface.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveEntity implements the ontology.Service interface.
func (s *Service) RetrieveEntity(key string) (schema.Entity, error) {
	uuidKey, err := uuid.Parse(key)
	if err != nil {
		return schema.Entity{}, err
	}
	u, err := s.Retrieve(uuidKey)
	return newEntity(u), err
}

func newEntity(u User) schema.Entity {
	e := schema.NewEntity(_schema, u.Username)
	schema.Set[uuid.UUID](e, "key", u.Key)
	schema.Set[string](e, "username", u.Username)
	return e
}
