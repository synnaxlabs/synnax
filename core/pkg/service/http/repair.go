// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
)

// repairRequest drops the header and query parameter rows with no name, which the
// Driver skipped. It moves a query written into path into params, decoded, so the
// Driver sends the same query in its own order and encoding. A query that repeats a
// name or does not decode stays in path, because params cannot hold it.
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
		rawName, rawValue, _ := strings.Cut(pair, "=")
		name, nameErr := url.QueryUnescape(rawName)
		value, valueErr := url.QueryUnescape(rawValue)
		taken := slices.ContainsFunc(
			moved,
			func(q QueryParam) bool { return q.Parameter == name },
		)
		if nameErr != nil || valueErr != nil || name == "" || taken {
			return
		}
		moved = append(moved, QueryParam{Parameter: name, Value: value})
	}
	*path, *params = base, moved
}

// reconcileRead repairs each endpoint of a read config that an older Core stored.
func reconcileRead(_ context.Context, _ gorp.Tx, c *ReadConfig) error {
	for i := range c.Endpoints {
		ep := &c.Endpoints[i]
		repairRequest(&ep.Path, &ep.Headers, &ep.QueryParams)
	}
	return nil
}

// reconcileWrite repairs each endpoint of a write config that an older Core stored.
func reconcileWrite(_ context.Context, _ gorp.Tx, c *WriteConfig) error {
	for i := range c.Endpoints {
		ep := &c.Endpoints[i]
		repairRequest(&ep.Path, &ep.Headers, &ep.QueryParams)
	}
	return nil
}
