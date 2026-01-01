// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"io"
	"iter"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
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

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.Uint64().Coerce(),
	"name":     zyn.String(),
	"type":     zyn.String(),
	"snapshot": zyn.Bool(),
})

func newResource(t Task) ontology.Resource {
	return core.NewResource(schema, OntologyID(t.Key), t.Name, t)
}

type change = changex.Change[Key, Task]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

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

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Task]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[Key, Task](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[Key, Task](s.cfg.DB).OpenNexter(ctx)
	return xiter.Map(n, newResource), closer, err
}
