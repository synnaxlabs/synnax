// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/gorp"
)

const ontologyType schema.Type = "group"

func OntologyID(k uuid.UUID) schema.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

type Group struct {
	Key  uuid.UUID
	Name string
}

var _ gorp.Entry[uuid.UUID] = Group{}

// GorpKey implements gorp.Entry.
func (c Group) GorpKey() uuid.UUID { return c.Key }

// SetOptions implements gorp.Entry.
func (c Group) SetOptions() []interface{} { return nil }

var _schema = &schema.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":  {Type: schema.UUID},
		"name": {Type: schema.String},
	},
}

func newResource(g Group) schema.Resource {
	r := schema.NewResource(_schema, OntologyID(g.Key), g.Name)
	schema.Set(r, "key", g.Key)
	schema.Set(r, "name", g.Name)
	return r
}
