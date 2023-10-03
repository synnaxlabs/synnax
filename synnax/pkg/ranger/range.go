// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Range (short for time range) is an interesting, user defined regions of time in a
// Synnax cluster. They act as a method for labeling and categorizing data.
type Range struct {
	tx gorp.Tx
	// Key is a unique identifier for the Range. If not provided on creation, a new one
	// will be generated.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the range. This name does not need to be unique.
	Name string `json:"name" msgpack:"name"`
	// TimeRange is the range of time occupied by the range.
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
}

var _ gorp.Entry[uuid.UUID] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []interface{} { return nil }

func (r Range) UseTx(tx gorp.Tx) Range { r.tx = tx; return r }

func (r Range) Get(ctx context.Context, key []byte) ([]byte, error) {
	var (
		res = keyValue{Range: r.Key, Key: key}
		err = gorp.NewRetrieve[[]byte, keyValue]().
			WhereKeys(res.GorpKey()).
			Entry(&res).
			Exec(ctx, r.tx)
	)
	return res.Value, err
}

func (r Range) Set(ctx context.Context, key, value []byte) error {
	return gorp.NewCreate[[]byte, keyValue]().
		Entry(&keyValue{Range: r.Key, Key: key, Value: value}).
		Exec(ctx, r.tx)
}

func (r Range) Delete(ctx context.Context, key []byte) error {
	return gorp.NewDelete[[]byte, keyValue]().
		WhereKeys(keyValue{Range: r.Key, Key: key}.GorpKey()).
		Exec(ctx, r.tx)
}

func (r Range) SetAlias(ctx context.Context, ch channel.Key, al string) error {
	exists, err := gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(ch).Exists(ctx, r.tx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "[range] - cannot alias non-existent channel %s", ch)
	}
	return gorp.NewCreate[string, alias]().
		Entry(&alias{Range: r.Key, Channel: ch, Alias: al}).
		Exec(ctx, r.tx)
}

func (r Range) ResolveAlias(ctx context.Context, al string) (channel.Key, error) {
	var res alias
	err := gorp.NewRetrieve[string, alias]().
		WhereKeys(alias{Range: r.Key, Alias: al}.GorpKey()).
		Entry(&res).
		Exec(ctx, r.tx)
	return res.Channel, err
}
