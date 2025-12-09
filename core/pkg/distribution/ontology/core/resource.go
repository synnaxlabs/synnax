// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// Type is the type of a specific ontology Resource. This type should be unique for each
// [Schema] in the cluster.
type Type string

// ZeroType is the zero type and should be assigned to any resource.
const ZeroType = Type("")

// String implements fmt.Stringer.
func (t Type) String() string { return string(t) }

// ID is a unique identifier for a Resource. An example:
//
//	userKey := Name{
//	    Name:  "748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
//	    Variant: "user",
//	}
//
// The key has two elements for several reasons. First, by storing the Type we know which
// Service to query for additional info on the Resource. Second, while a [ID.Key] may be
// unique for a particular resource (e.g. channel), it might not be unique across ALL
// resources. We need something universally unique across the entire delta cluster.
type ID struct {
	// Key is a string that uniquely identifies a Resource within its Type.
	Key string `json:"key" msgpack:"key"`
	// Type defines the type of Resource the Key refers to :). For example,
	// a channel is a Resource of type "channel". Key user is a Resource of type
	// "user".
	Type Type `json:"type" msgpack:"type"`
}

// Validate ensures that the given ID has both a Key and Type.
func (id ID) Validate() error {
	if id.Key == "" {
		return errors.Wrapf(validate.Error, "[resource] - key is required")
	}
	if id.Type == "" {
		return errors.Wrapf(validate.Error, "[resource] - type is required")
	}
	return nil
}

// String returns a string representation of the ID in the format "type:key".
func (id ID) String() string { return string(id.Type) + ":" + id.Key }

// IsZero returns true if the ID is the zero value (both Key and Type are empty).
func (id ID) IsZero() bool { return id.Key == "" && id.Type == "" }

// IsType returns true if the ID represents a type identifier (has a Type but no Key).
// Type IDs are used to identify resource types rather than specific resource instances.
func (id ID) IsType() bool { return id.Type != "" && id.Key == "" }

// ParseID parses the given string into an ID.
func ParseID(s string) (ID, error) {
	// We explicitly allow ontology id's that have multiple colons i.e.
	// 'foo:bar:baz' will be parsed as ID{type: 'foo', Key: 'bar:baz'}
	split := strings.SplitN(s, ":", 2)
	if len(split) != 2 {
		return ID{}, errors.Wrapf(validate.Error, "[ontology] - failed to parse id: %s", s)
	}
	if split[0] == "" {
		return ID{}, errors.Wrapf(validate.Error, "[ontology] - failed to parse id: %s (empty type)", s)
	}
	return ID{Type: Type(split[0]), Key: split[1]}, nil
}

// ParseIDs parses the given strings into keys.
func ParseIDs(s []string) ([]ID, error) {
	ids := make([]ID, 0, len(s))
	for _, id := range s {
		parsed, err := ParseID(id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, parsed)
	}
	return ids, nil
}

// IDs extracts the IDs from a slice of Resources.
func ResourceIDs(resources []Resource) []ID {
	ids := make([]ID, 0, len(resources))
	for _, r := range resources {
		ids = append(ids, r.ID)
	}
	return ids
}

// IDsToString converts a slice of IDs to a slice of their string representations.
func IDsToString(ids []ID) []string {
	strings := make([]string, 0, len(ids))
	for _, id := range ids {
		strings = append(strings, id.String())
	}
	return strings
}

// Resource represents an instance matching a [Schema] (think class and object in OOP).
type Resource struct {
	ID ID `json:"id" msgpack:"id"`
	// Name is a human-readable name for the entity.
	Name string `json:"name" msgpack:"name"`
	// Data is the data for the entity. Data must match [Schema.Fields].
	Data any `json:"data" msgpack:"data"`
	// schema is the schema that this entity matches.
	schema zyn.Schema
}

// Parse parses the Resource's Data field into the provided destination using the
// resource's schema. Returns an error if the data does not match the schema or
// cannot be parsed into the destination type.
func (r Resource) Parse(dest any) error {
	return r.schema.Parse(r.Data, dest)
}

type Change = change.Change[ID, Resource]

// BleveType returns the type of the entity for use search indexing,
// implementing the bleve.bleveClassifier interface.
func (r Resource) BleveType() string { return string(r.ID.Type) }

var _ gorp.Entry[ID] = Resource{}

// GorpKey implements gorp.Entry.
func (r Resource) GorpKey() ID { return r.ID }

// SetOptions implements gorp.Entry.
func (r Resource) SetOptions() []any { return nil }

// NewResource creates a new entity with the given schema and name and an empty set of
// field data. NewResource panics if the provided data value does not fit the ontology
// schema.
func NewResource(schema zyn.Schema, id ID, name string, data any) Resource {
	return Resource{
		schema: schema,
		ID:     id,
		Name:   name,
		Data:   lo.Must(schema.Dump(data)),
	}
}
