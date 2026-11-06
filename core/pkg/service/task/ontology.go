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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// ResourceID returns a unique ID for the task with the given key within the Synnax
// ontology.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeTask, Key: key.String()}
}

// ActionRecord returns the resource ID of the task that a sample of the action channel
// edits.
func ActionRecord(sample []byte) (resource.ID, error) {
	var scoped actions.Scoped[Key, jsontext.Value]
	if err := json.Unmarshal(sample, &scoped); err != nil {
		return resource.ID{}, err
	}
	return ResourceID(scoped.Key), nil
}

// ResourceIDs returns the resource IDs for the given keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

func ResourceIDsFromTasks(tasks []Task) []resource.ID {
	return lo.Map(tasks, func(task Task, _ int) resource.ID {
		return task.ResourceID()
	})
}

func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
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
	return ontology.NewResource(schema, t.ResourceID(), t.Name, t)
}

type change = xchange.Change[Key, Task]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() resource.Type { return resource.TypeTask }

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
		Key:     ResourceID(c.Key).String(),
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

func newSearchIndex() *gorp.SearchIndex[Key, Task] {
	return gorp.NewSearchIndex("search", func(e *Task) []string {
		return []string{e.Name, e.Type}
	})
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, s.search, ResourceID, q)
}
