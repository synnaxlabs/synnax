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
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns a unique ID for the task with the given key within the Synnax
// ontology.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTask, Key: key.String()}
}

// OntologyIDs returns the ontology IDs for the given keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

func OntologyIDsFromTasks(tasks []Task) []ontology.ID {
	return lo.Map(tasks, func(task Task, _ int) ontology.ID {
		return task.OntologyID()
	})
}

func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return uuid.Parse(id.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.UUID(),
	"name":     zyn.String(),
	"type":     zyn.String(),
	"snapshot": zyn.Bool(),
})

func newResource(t Task) ontology.Resource {
	return ontology.NewResource(schema, t.OntologyID(), t.Name, t)
}

type change = xchange.Change[Key, Task]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeTask }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var t Task
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&t).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(t), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Task]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, term, limit, OntologyID)
}
