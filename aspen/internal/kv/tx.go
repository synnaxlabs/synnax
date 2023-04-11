// Copyright 2023 Synnax Labs, Inc.
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
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/binary"
	kvx "github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
	"sync"
)

// tx is an aspen-managed key-value transaction. It's important to note that aspen
// does not support atomicity on transactions with lease cardinality greater than
// one i.e. if a transaction contains operations with different leaseholders, then
// the transaction is not guaranteed to be atomic. See https://github.com/synnaxlabs/synnax/issues/102
// for more details.
type tx struct {
	alamos.Instrumentation
	// Tx is the underlying key-value transaction. This transaction is not actually
	// applied, and simply serves as a cache for the operations that are applied.
	// It also serves all read operations.
	kvx.Tx
	digests []Digest
	lease   *leaseAllocator
	apply   func(reqs []TxRequest) error
}

var _ kvx.Tx = (*tx)(nil)

// Set implements kvx.Tx.
func (b *tx) Set(ctx context.Context, key, value []byte, options ...interface{}) error {
	lease, err := validateLeaseOption(options)
	if err != nil {
		return err
	}
	return b.applyOp(ctx, Operation{
		Key:         key,
		Value:       value,
		Variant:     Set,
		Leaseholder: lease,
	})
}

// Delete implements kvx.Tx.
func (b *tx) Delete(ctx context.Context, key []byte, _ ...interface{}) error {
	return b.applyOp(ctx, Operation{Key: key, Variant: Delete})
}

func (b *tx) applyOp(ctx context.Context, op Operation) error {
	var err error
	op, err = b.lease.allocate(ctx, op)
	if err != nil {
		return err
	}
	if op.Variant == Delete {
		if err := b.Tx.Delete(ctx, op.Key); err != nil {
			return err
		}
	} else {
		if err := b.Tx.Set(ctx, op.Key, op.Value); err != nil {
			return err
		}
	}
	op.Key = binary.MakeCopy(op.Key)
	b.digests = append(b.digests, op.Digest())
	return nil
}

func (b *tx) toRequests(ctx context.Context) ([]TxRequest, error) {
	dm := make(map[node.Key]TxRequest)
	for _, dig := range b.digests {
		op := dig.Operation()
		if op.Variant == Set {
			v, err := b.Tx.Get(ctx, dig.Key)
			if err != nil && err != kvx.NotFound {
				return nil, err
			}
			op.Value = binary.MakeCopy(v)
		}
		br, ok := dm[op.Leaseholder]
		if !ok {
			br.Operations = []Operation{op}
		} else {
			br.Operations = append(br.Operations, op)
		}
		br.Leaseholder = op.Leaseholder
		br.ctx, br.span = b.T.Trace(
			ctx,
			fmt.Sprintf("tx-%d", br.Leaseholder),
			alamos.DebugLevel,
		)
		dm[op.Leaseholder] = br
	}
	data := make([]TxRequest, 0, len(dm))
	for _, d := range dm {
		data = append(data, d)
	}
	return data, b.free()
}

func (b *tx) Close() error { return b.free() }

func (b *tx) Commit(ctx context.Context, _ ...interface{}) error {
	data, err := b.toRequests(ctx)
	if err != nil {
		return err
	}
	return b.apply(data)
}

func (b *tx) free() error {
	b.digests = nil
	return b.Tx.Close()
}

type TxRequest struct {
	Leaseholder node.Key
	Sender      node.Key
	Operations  []Operation
	ctx         context.Context
	doneF       func(err error)
	span        alamos.Span
}

func (br TxRequest) empty() bool { return len(br.Operations) == 0 }

func (br TxRequest) size() int { return len(br.Operations) }

func (br TxRequest) logArgs() []zap.Field {
	return []zap.Field{
		zap.Int("size", br.size()),
		zap.Uint64("leaseholder", uint64(br.Leaseholder)),
		zap.Uint64("sender", uint64(br.Sender)),
	}
}

func (br TxRequest) digests() []Digest {
	digests := make([]Digest, len(br.Operations))
	for i, op := range br.Operations {
		digests[i] = op.Digest()
	}
	return digests
}

func (br TxRequest) commitTo(db kvx.TxnFactory) (err error) {
	b := db.OpenTx()
	defer func() {
		br.Operations = nil
		if err != nil {
			err = b.Close()
		} else if _err := b.Commit(br.ctx); _err != nil {
			err = _err
		}
		br.done(err)
	}()
	for _, op := range br.Operations {
		if _err := op.apply(br.ctx, b); _err != nil {
			err = _err
			return err
		}
		if _err := op.Digest().apply(br.ctx, b); _err != nil {
			err = _err
			return err
		}
	}
	return err
}

func (br TxRequest) done(err error) {
	if br.doneF != nil {
		br.doneF(err)
	}
	if br.span != nil {
		br.span.End()
	}
}

func validateLeaseOption(maybeLease []interface{}) (node.Key, error) {
	lease := DefaultLeaseholder
	if len(maybeLease) == 1 {
		l, ok := maybeLease[0].(node.Key)
		if !ok {
			return 0, errors.New("[aspen] - Leaseholder option must be of type node.Key")
		}
		lease = l
	}
	return lease, nil
}

type txCoordinator struct {
	wg sync.WaitGroup
	mu struct {
		sync.Mutex
		err error
	}
}

func (bc *txCoordinator) done(err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err != nil {
		bc.mu.err = err
	}
	bc.wg.Done()
}

func (bc *txCoordinator) wait() error {
	bc.wg.Wait()
	return bc.mu.err
}

func (bc *txCoordinator) add(data *TxRequest) {
	bc.wg.Add(1)
	data.doneF = bc.done
}
