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

package task

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"strconv"
)

const OntologyType ontology.Type = "task"

func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(item Key, _ int) ontology.ID {
		return OntologyID(item)
	})
}

func OntologyIDsFromTasks(ts []Task) []ontology.ID {
	return lo.Map(ts, func(item Task, _ int) ontology.ID {
		return OntologyID(item.Key)
	})
}

func KeysFromOntologyIds(ids []ontology.ID) (keys []Key, err error) {
	keys = make([]Key, len(ids))
	for i, id := range ids {
		k, err := strconv.Atoi(id.Key)
		if err != nil {
			return nil, err
		}
		keys[i] = Key(k)
	}
	return keys, nil
}

var _schema = &ontology.Schema{
	Type: OntologyType,
	Fields: map[string]schema.Field{
		"key":  {Type: schema.Uint32},
		"name": {Type: schema.String},
		"type": {Type: schema.String},
	},
}

func newResource(t Task) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(t.Key), t.Name)
	schema.Set(e, "key", uint32(t.Key))
	schema.Set(e, "name", t.Name)
	schema.Set(e, "type", t.Type)
	return e
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[Key, Task]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k, err := strconv.Atoi(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var r Task
	err = s.NewRetrieve().WhereKeys(Key(k)).Entry(&r).Exec(ctx, tx)
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Task]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[Key, Task](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[Key, Task](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Task, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
