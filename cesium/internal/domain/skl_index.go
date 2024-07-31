// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type index struct {
	alamos.Instrumentation
	skl          *SkipList
	deleteLock   sync.RWMutex
	indexPersist *indexPersist
	persistHead  int
}

func (idx *index) overlap(ctx context.Context, tr telem.TimeRange) bool {
	_, ok := idx.skl.SearchLE(ctx, tr)
	return ok
}

func (idx *index) timeRange() telem.TimeRange {
	return idx.skl.TimeRange()
}

func (idx *index) insert(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "domain/index.insert")
	defer span.End()

	if p.fileKey == 0 {
		idx.L.DPanic("fileKey must be set")
		return span.Error(errors.New("inserted pointer cannot have key 0"))
	}

	if err := idx.skl.Insert(ctx, p); err != nil {
		return err
	}

	if persist {
		persistPointers := idx.indexPersist.prepare(idx.persistHead)
		return persistPointers()
	}

	return nil
}

func (idx *index) update(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "domain/index.update")
	defer span.End()

	if p.fileKey == 0 {
		idx.L.DPanic("fileKey must be set")
		return span.Error(errors.New("inserted pointer cannot have key 0"))
	}

	if err := idx.skl.Update(ctx, p); err != nil {
		return err
	}

	if persist {
		persistPointers := idx.indexPersist.prepare(idx.persistHead)
		return persistPointers()
	}

	return nil
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	last := idx.skl.Last()
	if last == nil {
		// Index is empty, everything is after last.
		return true
	}
	return ts.After(last.Ptr.End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	first := idx.skl.First()
	if first == nil {
		// Index is empty, everything is before first.
		return true
	}
	return ts.Before(first.Ptr.Start)
}

func (idx *index) searchLE(ctx context.Context, ts telem.TimeStamp) *Node {
	n, _ := idx.skl.SearchLE(ctx, ts.SpanRange(0))
	return n
}

func (idx *index) searchGE(ctx context.Context, ts telem.TimeStamp) *Node {
	n, _ := idx.skl.SearchLE(ctx, ts.SpanRange(0))
	return n
}

func (idx *index) close() error {
	idx.skl = nil
	return nil
}
