/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package device

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const OntologyType ontology.Type = "device"

func OntologyID(k string) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k}
}

func KeysFromOntologyIDs(ids []ontology.ID) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.Key
	}
	return keys
}

var _schema = &ontology.Schema{
	Type: OntologyType,
	Fields: map[string]schema.Field{
		"key":        {Type: schema.String},
		"name":       {Type: schema.String},
		"make":       {Type: schema.String},
		"model":      {Type: schema.String},
		"configured": {Type: schema.Bool},
		"location":   {Type: schema.String},
		"rack":       {Type: schema.Uint32},
	},
}

func newResource(r Device) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(r.Key), r.Name)
	schema.Set(e, "key", r.Key)
	schema.Set(e, "name", r.Name)
	schema.Set(e, "make", r.Make)
	schema.Set(e, "model", r.Model)
	schema.Set(e, "configured", r.Configured)
	schema.Set(e, "location", r.Location)
	schema.Set(e, "rack", uint32(r.Rack))
	return e
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[string, Device]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	var r Device
	err := s.NewRetrieve().WhereKeys(key).Entry(&r).Exec(ctx, tx)
	return newResource(r), err
}

func translateChange(c change) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Device]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[string, Device](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[string, Device](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Device, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
