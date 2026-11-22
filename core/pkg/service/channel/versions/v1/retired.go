// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"bytes"
	"context"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/set"
)

// retiredNames are the internal channels that the Core announced ontology and range
// alias changes on, and stored calculation statuses in, before v63.
var retiredNames = set.New(
	"sy_calculation_state",
	"sy_calculation_status",
	"sy_ontology_resource_set",
	"sy_ontology_resource_delete",
	"sy_ontology_relationship_set",
	"sy_ontology_relationship_delete",
	"sy_range_alias_set",
	"sy_range_alias_delete",
	"sy_range_alias_action_set",
)

// retiredPrefix starts the key of each staged retired channel. The rest of the key is
// the channel key.
const retiredPrefix = "sy_v63_retired_channel/"

// RetiredMigration stages the key of each stored channel that is internal and holds a
// retired name (RetiredKeys), so a later step deletes exactly those channels. It runs
// before the step that drops the internal flag.
var RetiredMigration = migrate.New(
	"v63_stage_retired_channels",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var channels []Channel
		if err := gorp.NewRetrieve[Key, Channel]().
			Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
				return c.Internal && retiredNames.Contains(c.Name), nil
			})).
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
		for _, c := range channels {
			if err := tx.Set(ctx, retiredKey(c.Key()), nil); err != nil {
				return err
			}
		}
		return nil
	},
)

func retiredKey(k Key) []byte { return []byte(retiredPrefix + k.String()) }

// RetiredKeys returns the key of each channel that RetiredMigration staged.
func RetiredKeys(r kv.Reader) (keys []Key, err error) {
	iter, err := r.OpenIterator(kv.IterPrefix([]byte(retiredPrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		raw := string(bytes.TrimPrefix(iter.Key(), []byte(retiredPrefix)))
		k, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, err
		}
		keys = append(keys, Key(k))
	}
	return keys, iter.Error()
}

// ClearRetired deletes the keys that RetiredMigration staged.
func ClearRetired(ctx context.Context, tx kv.ReadWriter) error {
	keys, err := RetiredKeys(tx)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err = tx.Delete(ctx, retiredKey(k)); err != nil {
			return err
		}
	}
	return nil
}
