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
	"fmt"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	kvx "github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
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
		Change:      kvx.Change{Key: key, Value: value, Variant: change.Set},
		Leaseholder: lease,
	})
}

// Delete implements kvx.Tx.
func (b *tx) Delete(ctx context.Context, key []byte, _ ...interface{}) error {
	op := Operation{Change: kvx.Change{Key: key, Variant: change.Delete}}
	return b.applyOp(ctx, op)
}

// Close implements kv.Tx.
func (b *tx) Close() error {
	b.digests = nil
	return b.Tx.Close()
}

// Commit implements kv.Tx.
func (b *tx) Commit(ctx context.Context, _ ...interface{}) error {
	if ctx == nil {
		b.L.DPanic("aspen encountered a nil context when committing transaction")
	}
	ctx, span := b.T.Prod(ctx, "tx-commit")
	data, err := b.toRequests(ctx)
	if err != nil {
		return span.EndWith(err)
	}
	return span.EndWith(b.apply(data))
}

func (b *tx) applyOp(ctx context.Context, op Operation) error {
	var err error
	op, err = b.lease.allocate(ctx, op)
	if err != nil {
		return err
	}
	if op.Variant == change.Delete {
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
		if op.Variant == change.Set {
			v, closer, err := b.Tx.Get(ctx, dig.Key)
			if err != nil {
				return nil, err
			}
			if errors.Is(err, kvx.NotFound) {
				zap.S().Error("[aspen] - operation not found when batching tx", zap.String("key", string(dig.Key)))
				continue
			}
			op.Value = binary.MakeCopy(v)
			if err = closer.Close(); err != nil {
				return nil, err
			}
		}
		br, ok := dm[op.Leaseholder]
		if !ok {
			br.Operations = []Operation{op}
		} else {
			br.Operations = append(br.Operations, op)
		}
		br.Leaseholder = op.Leaseholder
		br.Context, br.span = b.T.Debug(
			ctx,
			fmt.Sprintf("tx-%d", br.Leaseholder),
		)
		dm[op.Leaseholder] = br
	}
	data := make([]TxRequest, 0, len(dm))
	for _, d := range dm {
		data = append(data, d)
	}
	return data, nil
}

type TxRequest struct {
	// Context is the context for the transaction. This context is important for
	// cancellation and tracing, but is extremely easy to misuse. If you don't know
	// what you're doing, be careful when passing this context around.
	Context     context.Context
	Leaseholder node.Key
	Sender      node.Key
	Operations  []Operation
	doneF       func(err error)
	span        alamos.Span
}

func (tr TxRequest) empty() bool { return len(tr.Operations) == 0 }

func (tr TxRequest) size() int { return len(tr.Operations) }

func (tr TxRequest) logFields() []zap.Field {
	return []zap.Field{
		zap.Int("size", tr.size()),
		zap.Uint64("leaseholder", uint64(tr.Leaseholder)),
		zap.Uint64("sender", uint64(tr.Sender)),
	}
}

func (tr TxRequest) commitTo(db kvx.Atomic) (err error) {
	b := db.OpenTx()
	defer func() {
		tr.Operations = nil
		if err != nil {
			err = b.Close()
		} else if _err := b.Commit(tr.Context); _err != nil {
			err = _err
		}
		tr.done(err)
	}()
	for _, op := range tr.Operations {
		if _err := op.apply(tr.Context, b); _err != nil {
			err = _err
			return err
		}
		if _err := op.Digest().apply(tr.Context, b); _err != nil {
			err = _err
			return err
		}
	}
	return err
}

func (tr TxRequest) done(err error) {
	if tr.doneF != nil {
		tr.doneF(err)
	}
	if tr.span != nil {
		tr.span.End()
	}
}

func (tr TxRequest) reader() kvx.TxReader { return &txReader{ops: tr.Operations} }

func (tr TxRequest) digests() []Digest {
	return lo.Map(tr.Operations, func(o Operation, _ int) Digest { return o.Digest() })
}

func validateLeaseOption(maybeLease []interface{}) (node.Key, error) {
	lease := DefaultLeaseholder
	if len(maybeLease) == 1 {
		l, ok := maybeLease[0].(node.Key)
		if !ok {
			return 0, errors.New("[aspen] - Leaseholder option must be of type node.Name")
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
	if err != nil {
		bc.mu.Lock()
		bc.mu.err = errors.Combine(bc.mu.err, err)
		bc.mu.Unlock()
	}
	bc.wg.Done()
}

func (bc *txCoordinator) wait() error {
	bc.wg.Wait()
	// At this point no other processes are writing to bc.mu.err, so no need to
	// lock.
	return bc.mu.err
}

func (bc *txCoordinator) add(data *TxRequest) {
	bc.wg.Add(1)
	data.doneF = bc.done
}

type txReader struct {
	curr int
	ops  []Operation
}

var _ kvx.TxReader = (*txReader)(nil)

// Count implements kvx.TxReader.
func (r *txReader) Count() int { return len(r.ops) }

// Next implements kvx.TxReader.
func (r *txReader) Next(_ context.Context) (kvx.Change, bool) {
	if r.curr >= len(r.ops) {
		return kvx.Change{}, false
	}
	op := r.ops[r.curr]
	r.curr++
	return op.Change, true
}
