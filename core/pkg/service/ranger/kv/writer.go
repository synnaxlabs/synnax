// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create and delete key-value pairs.
type Writer struct {
	tx gorp.Tx
}

// Set sets a key-value pair on the specified range.
func (w Writer) Set(ctx context.Context, rng uuid.UUID, key, value string) error {
	return gorp.NewCreate[string, Pair]().
		Entry(&Pair{Range: rng, Key: key, Value: value}).
		Exec(ctx, w.tx)
}

// SetMany sets multiple key-value pairs on the specified range.
func (w Writer) SetMany(ctx context.Context, rng uuid.UUID, pairs []Pair) error {
	for i, p := range pairs {
		p.Range = rng
		pairs[i] = p
	}
	return gorp.NewCreate[string, Pair]().Entries(&pairs).Exec(ctx, w.tx)
}

// Delete deletes a key-value pair from the specified range.
// Delete is idempotent and will not return an error if the key does not exist.
func (w Writer) Delete(ctx context.Context, rng uuid.UUID, key string) error {
	return gorp.NewDelete[string, Pair]().
		WhereKeys(Pair{Range: rng, Key: key}.GorpKey()).
		Exec(ctx, w.tx)
}
