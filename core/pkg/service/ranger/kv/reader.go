// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Reader is used to retrieve key-value pairs.
type Reader struct {
	tx gorp.Tx
}

// Get retrieves a single key-value pair from the specified range.
func (r Reader) Get(ctx context.Context, rng uuid.UUID, key string) (string, error) {
	var (
		res = Pair{Range: rng, Key: key}
		err = gorp.NewRetrieve[string, Pair]().
			WhereKeys(res.GorpKey()).
			Entry(&res).
			Exec(ctx, r.tx)
	)
	if errors.Is(err, query.NotFound) {
		return "", errors.Wrapf(err, "key %s not found on range", key)
	}
	return res.Value, err
}

// GetMany retrieves multiple key-value pairs from the specified range.
func (r Reader) GetMany(ctx context.Context, rng uuid.UUID, keys []string) ([]Pair, error) {
	res := make([]Pair, 0, len(keys))
	tKeys := lo.Map(keys, func(k string, _ int) string { return Pair{Range: rng, Key: k}.GorpKey() })
	err := gorp.NewRetrieve[string, Pair]().
		WhereKeys(tKeys...).
		Entries(&res).
		Exec(ctx, r.tx)
	return res, err
}

// List retrieves all key-value pairs on the specified range.
func (r Reader) List(ctx context.Context, rng uuid.UUID) ([]Pair, error) {
	var res []Pair
	err := gorp.NewRetrieve[string, Pair]().
		Where(func(_ gorp.Context, kv *Pair) (bool, error) {
			return kv.Range == rng, nil
		}).
		Entries(&res).
		Exec(ctx, r.tx)
	return res, err
}
