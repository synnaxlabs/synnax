// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"strings"
)

// ID is a unique identifier for a Resource. An example:
//
//	userKey := ID{
//	    ID:  "748d31e2-5732-4cb5-8bc9-64d4ad51efe8",
//	    Type: "user",
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
func (k ID) Validate() error {
	if k.Key == "" {
		return errors.Newf("[resource] - key is required")
	}
	if k.Type == "" {
		return errors.Newf("[resource] - type is required")
	}
	return nil
}

// String returns a string representation of the Resource.
func (k ID) String() string { return string(k.Type) + ":" + k.Key }

// ParseID parses the given string into an ID.
func ParseID(s string) (ID, error) {
	split := strings.Split(s, ":")
	if len(split) != 2 {
		return ID{}, errors.Errorf("[ontology] - failed to parse id: %s", s)
	}
	return ID{Type: schema.Type(split[0]), Key: split[1]}, nil
}

// ParseIDs parses the given strings into IDs.
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

// Resource represents a resource with a unique [ID] in the ontology along with its
// [Entity] information.
type Resource struct {
	// ID is the unique identifier for the resource.
	ID ID `json:"id" msgpack:"id"`
	// Entity is the entity information for the resource.
	Entity Entity `json:"entity" msgpack:"entity"`
}

// GorpKey implements the gorp.Entry interface.
func (r Resource) GorpKey() ID { return r.ID }

// SetOptions implements the gorp.Entry interface.
func (r Resource) SetOptions() []interface{} { return nil }
