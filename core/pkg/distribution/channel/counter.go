// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/math"
)

type counter struct {
	wrap *kv.AtomicInt64Counter
}

func openCounter(ctx context.Context, db kv.ReadWriter, key []byte) (*counter, error) {
	wrap, err := kv.OpenCounter(ctx, db, key)
	return &counter{wrap: wrap}, err
}

func (c *counter) add(delta LocalKey) (LocalKey, error) {
	if c.wrap.Value()+int64(delta) > int64(math.MaxUint20) {
		return 0, errors.New("maximum number of channels created")
	}
	next, err := c.wrap.Add(int64(delta))
	return LocalKey(next), err
}
