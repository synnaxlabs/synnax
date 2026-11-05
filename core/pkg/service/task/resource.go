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
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of tasks. Searches and lists leave out
// internal tasks.
func (s *Service) Resources() *resource.TableProvider[Key, Task] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Task]{
		Table:  s.table,
		Search: s.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(t *Task) string { return t.Name },
		Hidden: func(t *Task) bool { return t.Internal },
	})
}

// ResourceID returns the resource ID of the task with the given key.
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
		return resource.ParseUUIDKey(id)
	})
}

func newSearchIndex() *gorp.SearchIndex[Key, Task] {
	return gorp.NewSearchIndex("search", func(e *Task) []string {
		return []string{e.Name, e.Type}
	})
}
