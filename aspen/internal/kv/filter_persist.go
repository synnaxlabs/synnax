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

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// filterPersist fuses version filtering and persistence into a single segment
// for the gossip-ingress path. It opens one transaction per inbound TxRequest,
// reads each digest inside that transaction, and writes accepted ops and their
// digests through the same transaction before committing. Because reads inside
// the transaction see prior writes from the same transaction, two duplicate
// ops in one TxRequest cannot both pass. Because the segment processes one
// TxRequest at a time on a single goroutine and the engine state cannot change
// between the digest read and the digest write of any single op, duplicate
// deliveries across TxRequests are also rejected. This is the structural
// guarantee that backs the exactly-once observable contract documented on
// DB.NewObservable.
//
// The local write path (versionAssigner -> persist) does not flow through
// filterPersist: locally-assigned versions are monotonic by construction, so
// the digest check would always pass and would burden single-node deployments
// with a Get they cannot benefit from.
type filterPersist struct {
	confluence.BatchSwitch[TxRequest, TxRequest]
	db         xkv.DB
	acceptedTo address.Address
	rejectedTo address.Address
	Config
}

func newFilterPersist(
	cfg Config,
	acceptedTo address.Address,
	rejectedTo address.Address,
) segment {
	s := &filterPersist{
		Config:     cfg,
		db:         cfg.Engine,
		acceptedTo: acceptedTo,
		rejectedTo: rejectedTo,
	}
	s.Switch = s._switch
	return s
}

func (fp *filterPersist) _switch(
	_ context.Context,
	b TxRequest,
	o map[address.Address]TxRequest,
) error {
	ctx, span := fp.T.Debug(b.Context, "tx-filter-persist")
	defer span.End()

	accepted := TxRequest{
		Sender:  b.Sender,
		doneF:   b.doneF,
		Context: b.Context,
		span:    b.span,
	}
	rejected := TxRequest{Sender: b.Sender, Context: b.Context, span: b.span}

	err := xkv.WithTx(ctx, fp.db, func(txn xkv.Tx) error {
		for _, op := range b.Operations {
			sup, supErr := supersedes(ctx, txn, op)
			if supErr != nil {
				fp.L.Error(
					"failed to read digest for gossiped op",
					zap.Error(supErr),
					zap.Binary("key", op.Key),
				)
			}
			if !sup {
				rejected.Operations = append(rejected.Operations, op)
				continue
			}
			if err := op.apply(ctx, txn); err != nil {
				return err
			}
			if err := op.Digest().apply(ctx, txn); err != nil {
				return err
			}
			accepted.Operations = append(accepted.Operations, op)
		}
		return nil
	})
	accepted.done(err)

	if err == nil && !accepted.empty() {
		o[fp.acceptedTo] = accepted
	}
	if !rejected.empty() {
		o[fp.rejectedTo] = rejected
	}
	return nil
}

// supersedes reports whether op should replace the digest stored for op.Key in
// r. An op supersedes when its version is strictly newer, or when versions tie
// and its leaseholder wins the higher-key tiebreak. A missing digest is treated
// as superseded. Any other read error is returned and treated as
// not-superseding by the caller, so a transient read failure cannot cause an op
// to overwrite stored state.
func supersedes(ctx context.Context, r xkv.Reader, op Operation) (bool, error) {
	dig, err := getDigestFromKV(ctx, r, op.Key)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return true, nil
		}
		return false, err
	}
	if op.Version.EqualTo(dig.Version) {
		return op.Leaseholder > dig.Leaseholder, nil
	}
	return op.Version.NewerThan(dig.Version), nil
}
