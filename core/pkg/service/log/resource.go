// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of logs.
func (s *Service) Resources() *resource.TableProvider[Key, Log] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Log]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(l *Log) string { return l.Name },
	})
}

// ResourceID returns the resource ID of the log with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeLog, Key: k.String()}
}

// ResourceIDs returns the resource IDs of the logs with keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID {
		return ResourceID(key)
	})
}

// ResourceIDsFromLogs returns the resource IDs of the logs.
func ResourceIDsFromLogs(logs []Log) []resource.ID {
	return lo.Map(logs, func(l Log, _ int) resource.ID { return l.ResourceID() })
}
