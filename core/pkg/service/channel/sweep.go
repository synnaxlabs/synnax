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
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	// keptDataStatusKey is the key of the status that names the data Sweep keeps.
	keptDataStatusKey = "channel_kept_data"
	// corruptIntentsStatusKey is the key of the status that reports damaged deletes.
	corruptIntentsStatusKey = "channel_corrupt_intents"
)

// sweep reconciles storage with the channel records, before any writer opens, and
// raises a warning status that names the data with no record and no delete. It clears
// the status when there is none.
func (s *Service) sweep(ctx context.Context) error {
	records, err := recordKeys(ctx, s.db)
	if err != nil {
		return err
	}
	kept, highest, err := sweepStorage(ctx, records, s.cfg.TS)
	if err != nil {
		return err
	}
	// A key that storage holds in any form must not go to a new channel, which would
	// merge with the old samples.
	if err = s.cfg.Channel.Reserve(ctx, Keys{Key(highest)}); err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.cfg.Status.NewWriter(tx)
		if err := setOrDelete(
			ctx,
			w,
			keptDataStatusKey,
			len(kept) > 0,
			func() status.Status[any] {
				return keptDataStatus(kept)
			},
		); err != nil {
			return err
		}
		corrupt := s.cfg.TS.CorruptIntents()
		return setOrDelete(
			ctx,
			w,
			corruptIntentsStatusKey,
			corrupt > 0,
			func() status.Status[any] {
				return corruptIntentsStatus(corrupt)
			},
		)
	})
}

// Purge removes from ts the data of keys: data that has no channel record in db and no
// delete. It makes the deletes durable intents before it drops the data, as a channel
// delete does. It returns the channels that still hold such data, so with no keys it
// lists them. It refuses a key that has a channel record or holds no such data, and
// then removes nothing. No Core may run on db or ts.
func Purge(
	ctx context.Context,
	db *gorp.DB,
	ts *cesium.DB,
	keys []Key,
) ([]cesium.KeptChannel, error) {
	records, err := recordKeys(ctx, db)
	if err != nil {
		return nil, err
	}
	recorded := set.New(records...)
	for _, k := range keys {
		if recorded.Contains(k) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"channel %d has a channel record. To remove its data, delete the channel",
				k,
			)
		}
	}
	kept, _, err := sweepStorage(ctx, records, ts)
	if err != nil {
		return nil, err
	}
	purged := set.New(keys...)
	keptKeys := set.New(lo.Map(kept, func(k cesium.KeptChannel, _ int) Key {
		return Key(k.Key)
	})...)
	for _, k := range keys {
		if !keptKeys.Contains(k) {
			return nil, errors.Wrapf(
				query.ErrNotFound,
				"channel %d holds no data without a channel record",
				k,
			)
		}
	}
	for _, k := range kept {
		if idx := Key(k.Index); purged.Contains(idx) && !purged.Contains(Key(k.Key)) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"channel %d is the index of channel %d, which also holds data with no "+
					"channel record. Purge both channels together",
				idx,
				k.Key,
			)
		}
	}
	if len(keys) > 0 {
		if err = ts.Intend(Keys(keys).Uint32()); err != nil {
			return nil, err
		}
		if err = ts.Drop(Keys(keys).Uint32()); err != nil {
			return nil, err
		}
	}
	return lo.Reject(kept, func(k cesium.KeptChannel, _ int) bool {
		return purged.Contains(Key(k.Key))
	}), nil
}

// recordKeys returns the key of every channel record in db.
func recordKeys(ctx context.Context, db *gorp.DB) (Keys, error) {
	var records []Channel
	if err := gorp.NewRetrieve[Key, Channel]().Entries(&records).
		Exec(ctx, db); err != nil {
		return nil, err
	}
	return KeysFromChannels(records), nil
}

// sweepStorage sweeps ts against records and returns the data it keeps and the highest
// key it holds.
func sweepStorage(
	ctx context.Context,
	records Keys,
	ts *cesium.DB,
) ([]cesium.KeptChannel, cesium.ChannelKey, error) {
	stored := lo.Reject(records, func(k Key, _ int) bool { return k.Free() })
	return ts.Sweep(ctx, stored.Uint32())
}

// setOrDelete sets the status that build returns when raised, and deletes the status of
// key otherwise.
func setOrDelete(
	ctx context.Context,
	w status.Writer,
	key string,
	raised bool,
	build func() status.Status[any],
) error {
	if !raised {
		return w.Delete(ctx, key)
	}
	stat := build()
	return w.Set(ctx, &stat)
}

// corruptIntentsStatus returns the warning status that reports n damaged deletes.
func corruptIntentsStatus(n int) status.Status[any] {
	return status.Status[any]{
		Key:     corruptIntentsStatusKey,
		Variant: status.VariantWarning,
		Message: "Found damaged channel deletes",
		Description: fmt.Sprintf(
			"The storage engine found %d damaged records of channel deletes. The "+
				"data of a deleted channel can stay on disk. The warning for data "+
				"with no channel then names it.",
			n,
		),
		Time: telem.Now(),
	}
}

// keptDataStatus returns the warning status that names kept.
func keptDataStatus(kept []cesium.KeptChannel) status.Status[any] {
	var (
		keys  = make([]string, len(kept))
		total telem.Size
	)
	for i, k := range kept {
		keys[i] = strconv.FormatUint(uint64(k.Key), 10)
		total += k.Size
	}
	subject, verb, has := "Channels "+joinAnd(keys), "hold", "have"
	if len(kept) == 1 {
		subject, verb, has = "Channel "+keys[0], "holds", "has"
	}
	return status.Status[any]{
		Key:     keptDataStatusKey,
		Variant: status.VariantWarning,
		Message: "Found data with no channel",
		Description: fmt.Sprintf(
			"%s %s %s of data but %s no channel record and no delete. To keep the "+
				"data, restore the records from a backup. To remove it, stop the "+
				"Core and run `synnax purge %s`.",
			subject,
			verb,
			formatSize(total),
			has,
			strings.Join(keys, " "),
		),
		Time: telem.Now(),
	}
}

// joinAnd joins items as prose: "a", "a and b", or "a, b, and c".
func joinAnd(items []string) string {
	switch len(items) {
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
}

// formatSize formats s in the largest decimal unit it reaches.
func formatSize(s telem.Size) string {
	switch {
	case s >= telem.Gigabyte:
		return fmt.Sprintf("%.1f GB", float64(s)/float64(telem.Gigabyte))
	case s >= telem.Megabyte:
		return fmt.Sprintf("%.1f MB", float64(s)/float64(telem.Megabyte))
	case s >= telem.Kilobyte:
		return fmt.Sprintf("%.1f kB", float64(s)/float64(telem.Kilobyte))
	}
	return fmt.Sprintf("%d B", s)
}
