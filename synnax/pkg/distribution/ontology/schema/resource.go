// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schema

import (
	"fmt"
	"github.com/synnaxlabs/x/validate"
	"strings"

	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

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
	Key string
	// Type defines the type of Resource the Key refers to :). For example,
	// a channel is a Resource of type "channel". Key user is a Resource of type
	// "user".
	Type Type
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

// String returns a string representation of the Resource.
func (id ID) String() string { return string(id.Type) + ":" + id.Key }

// IsZero true if the ID is the zero value for its type.
func (id ID) IsZero() bool { return id.Key == "" && id.Type == "" }

func (id ID) IsType() bool { return id.Type != "" && id.Key == "" }

// ParseID parses the given string into an ID.
func ParseID(s string) (ID, error) {
	split := strings.Split(s, ":")
	if len(split) != 2 {
		return ID{}, errors.Wrapf(validate.Error, "[ontology] - failed to parse id: %s", s)
	}
	return ID{Type: Type(split[0]), Key: split[1]}, nil
}

// ParseIDs parses the given strings into Keys.
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

type Data = map[string]any

// Resource represents an instance matching a [Schema] (think class and object in OOP).
type Resource struct {
	ID ID `json:"id" msgpack:"id"`
	// Schema is the schema that this entity matches.
	Schema *Schema `json:"schema" msgpack:"schema"`
	// Name is a human-readable name for the entity.
	Name string `json:"name" msgpack:"name"`
	// Data is the data for the entity. Data must match [Schema.Fields].
	Data Data `json:"data" msgpack:"data"`
}

func ResourceIDs(resources []Resource) []ID {
	ids := make([]ID, 0, len(resources))
	for _, r := range resources {
		ids = append(ids, r.ID)
	}
	return ids
}

type Change = change.Change[ID, Resource]

// BleveType returns the type of the entity for use search indexing,
// implementing the bleve.bleveClassifier interface.
func (r Resource) BleveType() string { return string(r.ID.Type) }

var _ gorp.Entry[ID] = Resource{}

// GorpKey implements gorp.Entry.
func (r Resource) GorpKey() ID { return r.ID }

// SetOptions implements gorp.Entry.
func (r Resource) SetOptions() []interface{} { return nil }

// Get is a strongly-typed getter for a [Resource] field value. Returns true if the
// value was found, false otherwise. Panics if the value is not of the asserted type (
// as defined in the type parameter).
func Get[V Value](d Resource, k string) (v V, ok bool) {
	rv, ok := d.Data[k]
	if !ok {
		return v, false
	}
	v, ok = rv.(V)
	if !ok {
		panic("[schema] - invalid field type")
	}
	return v, true
}

// Set is a strongly-typed setter for an [Resource] field value. Panics if the value is
// not of the asserted type (as defined in the type parameter) or if the field is not
// defined in the [Schema].
func Set[V Value](D Resource, k string, v V) {
	f, ok := D.Schema.Fields[k]
	if !ok {
		panic("[Schema] - field not found")
	}
	if !f.AssertValue(v) {
		panic(fmt.Sprintf("[Schema] - invalid value %v for field %s", v, k))
	}
	D.Data[k] = v
}

// NewResource creates a new entity with the given schema and name and an empty set of
// field data.
func NewResource(schema *Schema, id ID, name string) Resource {
	return Resource{
		Schema: schema,
		ID:     id,
		Name:   name,
		Data:   make(map[string]any),
	}
}
