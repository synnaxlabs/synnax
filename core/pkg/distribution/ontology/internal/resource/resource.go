// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// Type is the type of a specific Resource. This type should be unique for each service
// in the ontology.
type Type string

// String implements fmt.Stringer.
func (t Type) String() string { return string(t) }

// ID is a unique identifier for a Resource. An example:
//
//	userID := ID{
//	    Key:  "748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
//	    Type: "user",
//	}
//
// The ID has two elements for several reasons. First, by storing the Type we know which
// service to query for additional info on the Resource. Second, while a Key may be
// unique for a particular resource (e.g. channel), it might not be unique across all
// resources. We need something universally unique across the entire Synnax Core.
type ID struct {
	// Key is a string that uniquely identifies a Resource within its Type.
	Key string `json:"key" msgpack:"key"`
	// Type defines the type of Resource the Key refers to. For example, a channel is a
	// Resource of Type "channel". A user is a Resource of Type "user".
	Type Type `json:"type" msgpack:"type"`
}

// Validate ensures that the given ID has both a Key and Type.
func (id ID) Validate() error {
	if id.Key == "" {
		return errors.Wrapf(validate.ErrValidation, "[ontology.resource] - key is required")
	}
	if id.Type == "" {
		return errors.Wrapf(validate.ErrValidation, "[ontology.resource] - type is required")
	}
	return nil
}

// String returns a string representation of the ID in the format "Type:Key".
func (id ID) String() string { return string(id.Type) + ":" + id.Key }

// IsZero returns true if the ID is the zero value (both Key and Type are empty).
func (id ID) IsZero() bool { return id.Key == "" && id.Type == "" }

// IsType returns true if the ID represents a type identifier (has a Type but no Key).
// Type IDs are used to identify resource types rather than specific resource instances.
func (id ID) IsType() bool { return id.Type != "" && id.Key == "" }

// ParseID parses the given key into an ID.
func ParseID(key string) (ID, error) {
	// We explicitly allow IDs that have multiple colons, i.e. "foo:bar:baz" will be
	// parsed as ID{Type: "foo", Key: "bar:baz"}.
	split := strings.SplitN(key, ":", 2)
	if len(split) != 2 {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[ontology.resource] - failed to parse id: %s",
			key,
		)
	}
	if split[0] == "" {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[ontology.resource] - failed to parse id: %s (empty type)",
			key,
		)
	}
	return ID{Type: Type(split[0]), Key: split[1]}, nil
}

// ParseIDs parses the given keys into IDs.
func ParseIDs(keys []string) ([]ID, error) {
	ids := make([]ID, len(keys))
	var err error
	for i, key := range keys {
		if ids[i], err = ParseID(key); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// IDsToKeys converts a slice of IDs to a slice of their keys.
func IDsToKeys(ids []ID) []string {
	return lo.Map(ids, func(id ID, _ int) string { return id.String() })
}

// Resource represents an instance matching of a resource in the ontology.
type Resource struct {
	// ID is the unique identifier for the Resource.
	ID ID `json:"id" msgpack:"id"`
	// Name is a human-readable name for the Resource.
	Name string `json:"name" msgpack:"name"`
	// Data is the data for the Resource. Data must be parseable by the Resource's
	// schema.
	Data any `json:"data" msgpack:"data"`
	// schema is the schema that this Resource matches.
	schema zyn.Schema
}

// New creates a new Resource with the given schema, name, and data. New panics if the
// provided data value does not fit the Resource's schema.
func New(schema zyn.Schema, id ID, name string, data any) Resource {
	return Resource{
		schema: schema,
		ID:     id,
		Name:   name,
		Data:   lo.Must(schema.Dump(data)),
	}
}

// Parse parses the Resource's Data field into the provided destination using the
// Resource's schema. Parse returns an error if the data does not match the schema or
// cannot be parsed into the destination type.
func (r Resource) Parse(dest any) error { return r.schema.Parse(r.Data, dest) }

var _ gorp.Entry[ID] = Resource{}

// BleveType returns the type of the entity for use search indexing,
// implementing the bleve.bleveClassifier interface.
func (r Resource) BleveType() string { return r.ID.Type.String() }

// GorpKey implements gorp.Entry.
func (r Resource) GorpKey() ID { return r.ID }

// SetOptions implements gorp.Entry.
func (r Resource) SetOptions() []any { return nil }

// Change is a change to a Resource.
type Change = change.Change[ID, Resource]

// IDs extracts the IDs from a slice of Resources.
func IDs(resources []Resource) []ID {
	return lo.Map(resources, func(r Resource, _ int) ID { return r.ID })
}
