package ontology

import (
	"fmt"
	"github.com/synnaxlabs/x/gorp"
)

// RelationshipType is a string that uniquely identifies the type of a relationship
// between two resources. For example, a relationship of type "member" could indicate
// that a particular resource is a member of another resource. When defining relationship
// types, use the synnax [Relationship.From] is the [Relationship.Type] of [Relationship.To]
// pattern. For example, if a relationship of type "member" indicates that a particular
// the variable should be named MemberOf (i.e. From is a MemberOf To).
type RelationshipType string

const (
	// ParentOf indicates that a resource is the parent of another resource. When
	// examining a Relationship of type ParentOf, the From field will be the parent
	// and the to field will be the child i.e. (From is the ParentOf To).
	ParentOf RelationshipType = "parent"
)

type Relationship struct {
	// From, To are the IDs of the related resources.
	From, To ID
	// Type is the type of relationship between the two resources. For more information
	// on relationship types, see the [RelationshipType] documentation.
	Type RelationshipType
}

var _ gorp.Entry[string] = Relationship{}

// GorpKey implements the gorp.Entry interface.
func (r Relationship) GorpKey() string {
	return fmt.Sprintf("%s:%s:%s", r.From.String(), r.To.String(), r.Type)

}

// SetOptions implements the gorp.Entry interface.
func (r Relationship) SetOptions() []interface{} { return nil }
