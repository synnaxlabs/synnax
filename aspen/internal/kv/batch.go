package kv

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/binary"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/cockroachdb/errors"
	"sync"
)

type batch struct {
	kvx.Batch
	digests []Digest
	lease   *leaseAllocator
	apply   func(reqs []BatchRequest) error
}

func (b *batch) Set(key []byte, value []byte, maybeLease ...interface{}) error {
	lease, err := validateLeaseOption(maybeLease)
	if err != nil {
		return err
	}
	return b.applyOp(Operation{
		Key:         key,
		Value:       value,
		Variant:     Set,
		Leaseholder: lease,
	})
}

func (b *batch) Delete(key []byte) error {
	return b.applyOp(Operation{Key: key, Variant: Delete})
}

func (b *batch) applyOp(op Operation) error {
	var err error
	op, err = b.lease.allocate(op)
	if err != nil {
		return err
	}
	if op.Variant == Delete {
		if err := b.Batch.Delete(op.Key); err != nil {
			return err
		}
	} else {
		if err := b.Batch.Set(op.Key, op.Value); err != nil {
			return err
		}
	}
	op.Key = binary.MakeCopy(op.Key)
	b.digests = append(b.digests, op.Digest())
	return nil
}

func (b *batch) toRequests() ([]BatchRequest, error) {
	dm := make(map[node.ID]BatchRequest)
	for _, dig := range b.digests {
		op := dig.Operation()
		if op.Variant == Set {
			v, err := b.Get(dig.Key)
			if err != nil && err != kvx.NotFound {
				return nil, err
			}
			op.Value = binary.MakeCopy(v)
		}
		ob, ok := dm[op.Leaseholder]
		if !ok {
			ob.Operations = []Operation{op}
		} else {
			ob.Operations = append(ob.Operations, op)
		}
		ob.Leaseholder = op.Leaseholder
		dm[op.Leaseholder] = ob
	}
	data := make([]BatchRequest, 0, len(dm))
	for _, d := range dm {
		data = append(data, d)
	}
	return data, b.free()
}

func (b *batch) Close() error { return b.free() }

func (b *batch) Commit(opts ...interface{}) error {
	data, err := b.toRequests()
	if err != nil {
		return err
	}
	return b.apply(data)
}

func (b *batch) free() error {
	b.digests = nil
	return b.Batch.Close()
}

type BatchRequest struct {
	Leaseholder node.ID
	Sender      node.ID
	Operations  []Operation
	done        func(err error)
}

func (bd BatchRequest) empty() bool { return len(bd.Operations) == 0 }

func (bd BatchRequest) size() int { return len(bd.Operations) }

func (bd BatchRequest) digests() []Digest {
	digests := make([]Digest, len(bd.Operations))
	for i, op := range bd.Operations {
		digests[i] = op.Digest()
	}
	return digests
}

func (bd BatchRequest) commitTo(bw kvx.BatchWriter) error {
	var err error
	b := bw.NewBatch()
	defer func() {
		bd.Operations = nil
		if _err := b.Commit(); _err != nil {
			err = _err
		}
		if bd.done != nil {
			bd.done(err)
		}
	}()
	for _, op := range bd.Operations {
		if _err := op.apply(b); _err != nil {
			err = _err
			return err
		}
		if _err := op.Digest().apply(b); _err != nil {
			err = _err
			return err
		}
	}
	return err
}

func validateLeaseOption(maybeLease []interface{}) (node.ID, error) {
	lease := DefaultLeaseholder
	if len(maybeLease) == 1 {
		l, ok := maybeLease[0].(node.ID)
		if !ok {
			return 0, errors.New("[aspen] - Leaseholder option must be of type node.ID")
		}
		lease = l
	}
	return lease, nil
}

type batchCoordinator struct {
	wg sync.WaitGroup
	mu struct {
		sync.Mutex
		err error
	}
}

func (bc *batchCoordinator) done(err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err != nil {
		bc.mu.err = err
	}
	bc.wg.Done()
}

func (bc *batchCoordinator) wait() error {
	bc.wg.Wait()
	return bc.mu.err
}

func (bc *batchCoordinator) add(data *BatchRequest) {
	bc.wg.Add(1)
	data.done = bc.done
}
