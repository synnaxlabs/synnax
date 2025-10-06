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

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
	"go.uber.org/zap"
)

type versionFilter struct {
	Config
	memKV      xkv.DB
	acceptedTo address.Address
	rejectedTo address.Address
	confluence.BatchSwitch[TxRequest, TxRequest]
}

func newVersionFilter(cfg Config, acceptedTo address.Address, rejectedTo address.Address) segment {
	s := &versionFilter{Config: cfg, acceptedTo: acceptedTo, rejectedTo: rejectedTo, memKV: cfg.Engine}
	s.Switch = s._switch
	return s
}

func (vc *versionFilter) _switch(
	_ context.Context,
	b TxRequest,
	o map[address.Address]TxRequest,
) error {
	var (
		rejected = TxRequest{Sender: b.Sender, doneF: b.doneF, Context: b.Context, span: b.span}
		accepted = TxRequest{Sender: b.Sender, doneF: b.doneF, Context: b.Context, span: b.span}
	)
	ctx, span := vc.T.Debug(b.Context, "tx-filter")
	defer span.End()
	for _, op := range b.Operations {
		if vc.filter(ctx, op) {
			accepted.Operations = append(accepted.Operations, op)
		} else {
			rejected.Operations = append(rejected.Operations, op)
		}
	}
	if len(accepted.Operations) > 0 {
		o[vc.acceptedTo] = accepted
	}
	if len(rejected.Operations) > 0 {
		o[vc.rejectedTo] = rejected
	}
	return nil
}

func (vc *versionFilter) filter(ctx context.Context, op Operation) bool {
	dig, err := getDigestFromKV(ctx, vc.memKV, op.Key)
	if err != nil {
		dig, err = getDigestFromKV(ctx, vc.Engine, op.Key)
		if err != nil {
			return errors.Is(err, xkv.NotFound)
		}
	}
	// If the versions of the operation are equal, we select a winning operation
	// based the which leasehold is higher.
	if op.Version.EqualTo(dig.Version) {
		return op.Leaseholder > dig.Leaseholder
	}
	return op.Version.NewerThan(dig.Version)
}

func getDigestFromKV(ctx context.Context, kve xkv.DB, key []byte) (Digest, error) {
	dig := Digest{}
	key, err := digestKey(key)
	if err != nil {
		return dig, err
	}
	b, closer, err := kve.Get(ctx, key)
	if err != nil {
		return dig, err
	}
	err = codec.Decode(ctx, b, &dig)
	err = errors.Combine(err, closer.Close())
	return dig, err
}

const versionCounterKey = "ver"

type versionAssigner struct {
	Config
	counter *xkv.AtomicInt64Counter
	confluence.LinearTransform[TxRequest, TxRequest]
}

func newVersionAssigner(ctx context.Context, cfg Config) (segment, error) {
	c, err := xkv.OpenCounter(ctx, cfg.Engine, []byte(versionCounterKey))
	v := &versionAssigner{Config: cfg, counter: c}
	v.Transform = v.assign
	return v, err
}

func (va *versionAssigner) assign(_ context.Context, br TxRequest) (TxRequest, bool, error) {
	latestVer := va.counter.Value()
	if _, err := va.counter.Add(int64(br.size())); err != nil {
		va.L.Error("failed to assign version", zap.Error(err))
		return TxRequest{}, false, nil
	}
	for i := range br.Operations {
		br.Operations[i].Version = version.Counter(latestVer + int64(i) + 1)
	}
	return br, true, nil
}
