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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return key.OntologyID() })
}

func OntologyIDsFromTasks(tasks []Task) []ontology.ID {
	return lo.Map(tasks, func(task Task, _ int) ontology.ID {
		return task.OntologyID()
	})
}

func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		k, err := strconv.ParseUint(id.Key, 10, 64)
		if err != nil {
			return 0, err
		}
		return Key(k), nil
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.Uint64().Coerce(),
	"name":     zyn.String(),
	"type":     zyn.String(),
	"snapshot": zyn.Bool(),
})

func newResource(t Task) ontology.Resource {
	return ontology.NewResource(schema, t.OntologyID(), t.Name, t)
}

type change = xchange.Change[Key, Task]

var (
	_ ontology.Service      = (*Service)(nil)
	_ search.Service        = (*Service)(nil)
	_ search.FieldsProvider = (*Service)(nil)
)

func (*Service) Type() ontology.ResourceType { return ontology.ResourceTypeTask }

// SearchableFields implements ontology.SearchableFieldsProvider.
func (*Service) SearchableFields() []string { return []string{"type"} }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := strconv.Atoi(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var t Task
	if err = s.NewRetrieve().
		Where(MatchKeys(Key(k))).
		Entry(&t).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(t), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     c.Key.OntologyID().String(),
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

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
