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
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// tx is an Aspen-managed key-value transaction. It's important to note that Aspen does
// not support atomicity on transactions with lease cardinality greater than one i.e.,
// if a transaction contains operations with different leaseholders, then the
// transaction is not guaranteed to be atomic. See
// https://github.com/synnaxlabs/synnax/issues/102 for more details.
type tx struct {
	// Tx is the underlying key-value transaction. This transaction is not actually
	// applied, and simply serves as a cache for the operations that are applied. It
	// also serves all read operations.
	kv.Tx
	lease *leaseAllocator
	apply func([]TxRequest) error
	alamos.Instrumentation
	digests []Digest
}

var _ kv.Tx = (*tx)(nil)

// Set implements kv.Tx.
func (t *tx) Set(ctx context.Context, key, value []byte, options ...any) error {
	lease, err := validateLeaseOption(options)
	if err != nil {
		return err
	}
	return t.applyOp(ctx, Operation{
		Change:      kv.Change{Key: key, Value: value, Variant: change.VariantSet},
		Leaseholder: lease,
	})
}

// Delete implements xkv.Tx.
func (t *tx) Delete(ctx context.Context, key []byte, _ ...any) error {
	op := Operation{Change: kv.Change{Key: key, Variant: change.VariantDelete}}
	return t.applyOp(ctx, op)
}

// Close implements kv.Tx.
func (t *tx) Close() error { t.digests = nil; return t.Tx.Close() }

// Commit implements kv.Tx.
func (t *tx) Commit(ctx context.Context, _ ...any) error {
	if ctx == nil {
		t.L.DPanic("Aspen encountered a nil context when committing a transaction")
	}
	ctx, span := t.T.Prod(ctx, "tx-commit")
	data, err := t.toRequests(ctx)
	if err != nil {
		return span.EndWith(err)
	}
	return span.EndWith(t.apply(data))
}

func (t *tx) applyOp(ctx context.Context, op Operation) error {
	var err error
	op, err = t.lease.allocate(ctx, op)
	if err != nil {
		return err
	}
	if op.Variant == change.VariantDelete {
		if err := t.Tx.Delete(ctx, op.Key); err != nil {
			return err
		}
	} else {
		if err := t.Tx.Set(ctx, op.Key, op.Value); err != nil {
			return err
		}
	}
	op.Key = bytes.Clone(op.Key)
	t.digests = append(t.digests, op.Digest())
	return nil
}

func (t *tx) toRequests(ctx context.Context) ([]TxRequest, error) {
	dm := make(map[node.Key]TxRequest)
	for _, dig := range t.digests {
		op := dig.Operation()
		if op.Variant == change.VariantSet {
			v, closer, err := t.Get(ctx, dig.Key)
			if errors.Is(err, query.ErrNotFound) {
				// A tx has a single writer, so a set key missing from the buffer can
				// only mean a later delete in the same tx superseded it; only the
				// delete needs to batch.
				continue
			}
			if err != nil {
				return nil, err
			}
			op.Value = bytes.Clone(v)
			if err = closer.Close(); err != nil {
				return nil, err
			}
		}
		req, ok := dm[op.Leaseholder]
		if !ok {
			req.Operations = []Operation{op}
		} else {
			req.Operations = append(req.Operations, op)
		}
		req.Leaseholder = op.Leaseholder
		req.Context, req.span = t.T.Debug(
			ctx,
			fmt.Sprintf("tx-%d", req.Leaseholder),
		)
		dm[op.Leaseholder] = req
	}
	return lo.MapToSlice(dm, func(k node.Key, r TxRequest) TxRequest { return r }), nil
}

type TxRequest struct {
	// Context is the context for the transaction. This context is important for
	// cancellation and tracing, but is extremely easy to misuse. If you don't know what
	// you're doing, be careful when passing this context around.
	Context     context.Context
	span        alamos.Span
	doneF       func(error)
	Operations  []Operation
	Leaseholder node.Key
	Sender      node.Key
}

func (tr TxRequest) empty() bool { return len(tr.Operations) == 0 }

func (tr TxRequest) size() int { return len(tr.Operations) }

func (tr TxRequest) commitTo(db kv.Atomic) (err error) {
	tx := db.OpenTx()
	defer func() {
		tr.Operations = nil
		if err != nil {
			err = tx.Close()
		} else if commitErr := tx.Commit(tr.Context); commitErr != nil {
			err = commitErr
		}
		tr.done(err)
	}()
	for _, op := range tr.Operations {
		if applyErr := op.apply(tr.Context, tx); applyErr != nil {
			err = applyErr
			return err
		}
		if applyErr := op.Digest().apply(tr.Context, tx); applyErr != nil {
			err = applyErr
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

func extractOpChange(op Operation) kv.Change { return op.Change }

func (tr TxRequest) reader() kv.TxReader {
	return iter.Map(slices.Values(tr.Operations), extractOpChange)
}

func (tr TxRequest) digests() []Digest {
	return lo.Map(tr.Operations, func(o Operation, _ int) Digest { return o.Digest() })
}

func validateLeaseOption(maybeLease []any) (node.Key, error) {
	lease := nodeKeyDefaultLeaseholder
	if len(maybeLease) == 1 {
		l, ok := maybeLease[0].(node.Key)
		if !ok {
			return 0, errors.New("leaseholder option must be of type node.Key")
		}
		lease = l
	}
	return lease, nil
}

type txCoordinator struct {
	mu struct {
		err error
		sync.Mutex
	}
	wg sync.WaitGroup
}

func (tc *txCoordinator) done(err error) {
	if err != nil {
		tc.mu.Lock()
		tc.mu.err = errors.Combine(tc.mu.err, err)
		tc.mu.Unlock()
	}
	tc.wg.Done()
}

func (tc *txCoordinator) wait() error {
	tc.wg.Wait()
	// At this point no other processes are writing to tc.mu.err, so no need to lock.
	return tc.mu.err
}

func (tc *txCoordinator) add(data *TxRequest) { tc.wg.Add(1); data.doneF = tc.done }
