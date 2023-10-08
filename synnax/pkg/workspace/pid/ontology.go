// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pid

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const ontologyType ontology.Type = "pid"

func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

func KeysFromOntologyIds(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":      {Type: schema.String},
		"name":     {Type: schema.String},
		"snapshot": {Type: schema.Bool},
	},
}

func newResource(pid PID) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(pid.Key), pid.Name)
	schema.Set(e, "key", pid.Key.String())
	schema.Set(e, "name", pid.Name)
	schema.Set(e, "snapshot", pid.Snapshot)
	return e
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, PID]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (schema.Resource, error) {
	k := uuid.MustParse(key)
	var pid PID
	err := s.NewRetrieve().WhereKeys(k).Entry(&pid).Exec(ctx, tx)
	return newResource(pid), err
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, PID]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, PID](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, PID](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[PID, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
