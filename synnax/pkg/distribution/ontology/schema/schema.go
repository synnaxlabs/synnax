// Package schema implements a method for defining, creating, and query dynamically typed
// entities (think dynamically typed structs). It aims to provide as much flexibility as possible
// while still providing strongly typed getters and setters for the entity fields.
package schema

// Type is the type of an [Entity]/[Schema]. This type should be unique for each
// [Schema] in the cluster. in the cluster. in the cluster. in the cluster.
type Type string

// Schema represents a dynamically defined schema for an arbitrary entity. This can be
// though of as a dynamically defined struct that allows entities of different types
// to be assembled into composite data structures (such as an ontology).
type Schema struct {
	// Type is the type of the entity. This type should be unique across all schemas
	// in the cluster.
	Type Type `json:"type" msgpack:"type"`
	// Fields is a map of field names to field types.
	Fields map[string]Field `json:"fields" msgpack:"fields"`
}
