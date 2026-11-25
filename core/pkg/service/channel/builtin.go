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
	"fmt"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// builtInSet holds the keys of the built-in channels, shaped so that a key needs one
// bit test or two compares: a bit for each local key of a free built-in channel, and
// the few leased built-in channels in ascending order.
type builtInSet struct {
	free   []uint64
	leased []Key
}

func (b *builtInSet) contains(k Key) bool {
	if k.Free() {
		local := uint32(k.LocalKey())
		word := int(local / 64)
		return word < len(b.free) && b.free[word]&(1<<(local%64)) != 0
	}
	n := len(b.leased)
	if n == 0 || k < b.leased[0] || k > b.leased[n-1] {
		return false
	}
	_, found := slices.BinarySearch(b.leased, k)
	return found
}

// with returns a copy of b that holds k.
func (b *builtInSet) with(k Key) *builtInSet {
	next := &builtInSet{free: b.free, leased: b.leased}
	if !k.Free() {
		if i, found := slices.BinarySearch(b.leased, k); !found {
			next.leased = slices.Insert(slices.Clone(b.leased), i, k)
		}
		return next
	}
	local := uint32(k.LocalKey())
	word := int(local / 64)
	next.free = slices.Clone(b.free)
	if word >= len(next.free) {
		next.free = append(next.free, make([]uint64, word+1-len(next.free))...)
	}
	next.free[word] |= 1 << (local % 64)
	return next
}

// IsBuiltIn reports whether a Core service declared the channel with key k for its own
// work through Writer.Declare.
func (s *Service) IsBuiltIn(k Key) bool { return s.builtIns.Load().contains(k) }

// Declare finds the channel named ch.Name or creates ch, and makes its key a built-in
// channel when the transaction commits. A channel of that name at the root, with the
// data type and virtual flag of ch, becomes the built-in, unless the holder of one of
// its uses computes it. Declare renames any other channel of that name with the first
// free suffix (_2, _3), writes a warning status for it, and stores the rename for its
// text pass (Writer.TakeTextRenames), since the holders of text are not open yet.
// Declare fills ch with the stored channel. It returns an error when another built-in
// holds the name.
func (w Writer) Declare(ctx context.Context, ch *Channel) error {
	var holders []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(ch.Name)).
		Entries(&holders).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	match := slices.IndexFunc(holders, func(h Channel) bool {
		return h.Group == nil && h.DataType == ch.DataType && h.Virtual == ch.Virtual
	})
	if match >= 0 {
		computed, err := w.computed(ctx, holders[match].Key())
		if err != nil {
			return err
		}
		if computed {
			match = -1
		}
	}
	for i, h := range holders {
		if i == match {
			continue
		}
		if err := w.rehome(ctx, h); err != nil {
			return err
		}
	}
	if match >= 0 {
		*ch = holders[match]
	} else if err := w.Create(ctx, ch, func(o *createOptions) {
		o.builtIn = true
	}); err != nil {
		return err
	}
	key := ch.Key()
	gorp.BeforeCommit(w.tx, func(context.Context) error {
		w.svc.mu.Lock()
		defer w.svc.mu.Unlock()
		w.svc.builtIns.Store(w.svc.builtIns.Load().with(key))
		w.svc.mu.externalNonVirtualSet.Remove(key)
		return nil
	})
	return nil
}

// computed reports whether the holder of a use of the channel with key computes it.
func (w Writer) computed(ctx context.Context, key Key) (bool, error) {
	if w.svc.cfg.Computes == nil {
		return false, nil
	}
	uses, err := w.svc.Uses(ctx, w.tx, []Key{key})
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(uses[0], w.svc.cfg.Computes), nil
}

// rehome renames h, a channel that holds the name of a built-in, with the first free
// suffix and writes a warning status that says so.
func (w Writer) rehome(ctx context.Context, h Channel) error {
	if w.svc.IsBuiltIn(h.Key()) {
		return errors.Newf(
			"built-in channel %q has another data type or virtual flag",
			h.Name,
		)
	}
	name, err := w.freeName(ctx, h.Name)
	if err != nil {
		return err
	}
	if err = w.Rename(ctx, h.Key(), name, false); err != nil {
		return err
	}
	if err = w.storeTextRename(ctx, h.Key(), h.Name); err != nil {
		return err
	}
	return w.svc.cfg.Status.NewWriter(w.tx).Set(ctx, &status.Status[any]{
		Name:    name,
		Variant: status.VariantWarning,
		Message: "Renamed channel",
		Description: fmt.Sprintf(
			"The Core needs the name %s for a built-in channel, so the channel %s is "+
				"now %s.",
			h.Name,
			h.Name,
			name,
		),
		Time: telem.Now(),
	})
}

// freeName returns name with the first suffix (_2, _3) that no channel holds.
func (w Writer) freeName(ctx context.Context, name string) (string, error) {
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", name, i)
		taken, err := w.svc.newRetrieve().
			Where(MatchNames(candidate)).
			Exists(ctx, w.tx)
		if err != nil || !taken {
			return candidate, err
		}
	}
}
