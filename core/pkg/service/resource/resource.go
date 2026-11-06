// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource

import (
	"context"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ParseID parses the given key into an ID.
func ParseID(key string) (ID, error) {
	// We explicitly allow IDs that have multiple colons, i.e. "foo:bar:baz" will be
	// parsed as ID{Type: "foo", Key: "bar:baz"}.
	split := strings.SplitN(key, ":", 2)
	if len(split) != 2 {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[resource] - failed to parse id: %s",
			key,
		)
	}
	if split[0] == "" {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[resource] - failed to parse id: %s (empty type)",
			key,
		)
	}
	return ID{Type: Type(split[0]), Key: split[1]}, nil
}

// ParseIDs parses the given keys into IDs.
func ParseIDs(keys []string) ([]ID, error) {
	return lo.MapErr(keys, func(key string, _ int) (ID, error) { return ParseID(key) })
}

// IDsToKeys converts a slice of IDs to a slice of their string representations.
func IDsToKeys(ids []ID) []string {
	return lo.Map(ids, func(id ID, _ int) string { return id.String() })
}

// Dependents returns the ID of every record that references the entry of table with key,
// as seen by tx. It blocks and fails as gorp.Table.Dependents does.
func Dependents[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	table *gorp.Table[K, E],
	key K,
) ([]ID, error) {
	deps, err := table.Dependents(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	return lo.Map(deps, func(d gorp.Target, _ int) ID {
		return ID{Type: Type(d.Table), Key: d.Key}
	}), nil
}
