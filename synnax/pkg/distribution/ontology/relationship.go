package ontology

import (
	"fmt"
)

type RelationshipType string

const (
	Parent RelationshipType = "parent"
)

type Relationship struct {
	From ID
	To   ID
	Type RelationshipType
}

func (r Relationship) GorpKey() string {
	return fmt.Sprintf("%s:%s:%s", r.From.String(), r.To.String(), r.Type)

}
func (r Relationship) SetOptions() []interface{} { return nil }
