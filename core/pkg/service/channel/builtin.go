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
	"maps"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// IsBuiltIn reports whether a Core service declared the channel with key k for its own
// work through Writer.Declare.
func (s *Service) IsBuiltIn(k Key) bool { return s.builtIns.Load().Contains(k) }

// BuiltIns returns the keys of the channels that Core services declared, in ascending
// order.
func (s *Service) BuiltIns() Keys {
	keys := Keys(s.builtIns.Load().Slice())
	slices.Sort(keys)
	return keys
}

// Declare finds the channel named ch.Name or creates ch, and makes its key a built-in
// channel when the transaction commits. A channel of that name at the root, with the
// data type and virtual flag of ch, becomes the built-in. Declare renames any other
// channel of that name with the first free suffix (_2, _3) and writes a warning status
// for it. Declare fills ch with the stored channel. It returns an error when another
// built-in holds the name, or when ch is calculated.
func (w Writer) Declare(ctx context.Context, ch *Channel) error {
	if ch.IsCalculated() {
		return errors.Wrapf(
			validate.ErrValidation,
			"built-in channel %q can't be calculated",
			ch.Name,
		)
	}
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
		next := maps.Clone(*w.svc.builtIns.Load())
		next.Add(key)
		w.svc.builtIns.Store(&next)
		w.svc.mu.externalNonVirtualSet.Remove(key)
		return nil
	})
	return nil
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
