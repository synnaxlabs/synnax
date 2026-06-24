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

	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
	"go.uber.org/zap"
)

func getDigestFromKV(ctx context.Context, r xkv.Reader, key []byte) (Digest, error) {
	dig := Digest{}
	key, err := digestKey(key)
	if err != nil {
		return dig, err
	}
	b, closer, err := r.Get(ctx, key)
	if err != nil {
		return dig, err
	}
	err = codec.Decode(ctx, b, &dig)
	err = errors.Combine(err, closer.Close())
	return dig, err
}

const versionCounterKey = "ver"

type versionAssigner struct {
	confluence.LinearTransform[TxRequest, TxRequest]
	counter *xkv.AtomicInt64Counter
	Config
}

func newVersionAssigner(ctx context.Context, cfg Config) (segment, error) {
	c, err := xkv.NewCounter(ctx, cfg.Engine, []byte(versionCounterKey))
	v := &versionAssigner{Config: cfg, counter: c}
	v.Transform = v.assign
	return v, err
}

func (va *versionAssigner) assign(ctx context.Context, br TxRequest) (TxRequest, bool, error) {
	latestVer := va.counter.Value()
	if _, err := va.counter.Add(ctx, int64(br.size())); err != nil {
		va.L.Error("failed to assign version", zap.Error(err))
		return TxRequest{}, false, nil
	}
	for i := range br.Operations {
		br.Operations[i].Version = version.Counter(latestVer + int64(i) + 1)
	}
	return br, true, nil
}
