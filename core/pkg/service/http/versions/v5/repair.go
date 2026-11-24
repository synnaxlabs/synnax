// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
)

// repairRequest drops the header and query parameter rows with no name, which the
// Driver skipped. It moves a query written into path into params, decoded, so the
// Driver sends the same query in its own order and encoding. A query that repeats a
// name, has a name with no '=', or does not decode stays in path, because params
// cannot hold it.
func repairRequest(path *string, headers *[]Header, params *[]QueryParam) {
	*headers = slices.DeleteFunc(*headers, func(h Header) bool { return h.Name == "" })
	*params = slices.DeleteFunc(
		*params,
		func(q QueryParam) bool { return q.Parameter == "" },
	)
	base, query, found := strings.Cut(*path, "?")
	if !found {
		return
	}
	moved := slices.Clone(*params)
	for pair := range strings.SplitSeq(query, "&") {
		if pair == "" {
			continue
		}
		rawName, rawValue, assigned := strings.Cut(pair, "=")
		name, nameErr := url.QueryUnescape(rawName)
		value, valueErr := url.QueryUnescape(rawValue)
		taken := slices.ContainsFunc(
			moved,
			func(q QueryParam) bool { return q.Parameter == name },
		)
		if !assigned || nameErr != nil || valueErr != nil || name == "" || taken {
			return
		}
		moved = append(moved, QueryParam{Parameter: name, Value: value})
	}
	*path, *params = base, moved
}

// RepairReadConfig repairs the requests of each endpoint of c, which an older Core
// may have stored before the header, query, and path rules.
func RepairReadConfig(c *ReadConfig) {
	for i := range c.Endpoints {
		ep := &c.Endpoints[i]
		repairRequest(&ep.Path, &ep.Headers, &ep.QueryParams)
	}
}

// RepairWriteConfig repairs the requests of each endpoint of c, which an older Core
// may have stored before the header, query, and path rules.
func RepairWriteConfig(c *WriteConfig) {
	for i := range c.Endpoints {
		ep := &c.Endpoints[i]
		repairRequest(&ep.Path, &ep.Headers, &ep.QueryParams)
	}
}

// repairMigration returns a migration that applies repair to each stored config of
// type E.
func repairMigration[E gorp.Entry[uuid.UUID]](
	key string,
	repair func(*E),
) gorp.Migration {
	return migrate.New(
		key,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var cfgs []E
			if err := gorp.NewRetrieve[uuid.UUID, E]().
				Entries(&cfgs).
				Exec(ctx, tx); err != nil || len(cfgs) == 0 {
				return err
			}
			for i := range cfgs {
				repair(&cfgs[i])
			}
			return gorp.WrapWriter[uuid.UUID, E](tx).Set(ctx, cfgs...)
		},
	)
}

// ReadRepairMigration repairs the requests of each stored read config.
var ReadRepairMigration = repairMigration("v64_http_read_repair_rows", RepairReadConfig)

// WriteRepairMigration repairs the requests of each stored write config.
var WriteRepairMigration = repairMigration(
	"v64_http_write_repair_rows",
	RepairWriteConfig,
)
