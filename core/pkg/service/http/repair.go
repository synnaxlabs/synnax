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
	"regexp"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
)

// unreserved matches a query name or value that the Driver sends as it is.
var unreserved = regexp.MustCompile("^[A-Za-z0-9._~-]*$")

// repairRequest drops the header and query parameter rows with no name, which the
// Driver skipped. It moves a query written into path into params when the Driver then
// sends the same URL: the query parameters were empty, and each name is unique,
// sorted, and unreserved, as is each value.
func repairRequest(path *string, headers *[]Header, params *[]QueryParam) {
	*headers = slices.DeleteFunc(*headers, func(h Header) bool { return h.Name == "" })
	*params = slices.DeleteFunc(
		*params,
		func(q QueryParam) bool { return q.Parameter == "" },
	)
	base, query, found := strings.Cut(*path, "?")
	if !found || query == "" || len(*params) > 0 {
		return
	}
	var moved []QueryParam
	for pair := range strings.SplitSeq(query, "&") {
		name, value, ok := strings.Cut(pair, "=")
		if !ok || name == "" || !unreserved.MatchString(name) ||
			!unreserved.MatchString(value) ||
			(len(moved) > 0 && name <= moved[len(moved)-1].Parameter) {
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
