// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/identifier"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// ErrInvalidVariant signals a variant string outside xstatus.AllowedVariants.
var ErrInvalidVariant = errors.New("invalid status variant")

// SetByKeyOrName upserts by name or updates by UUID key (returns query.ErrNotFound
// on miss). Both paths wrap retrieve + write in WithTx so a concurrent delete
// can't turn the write into a silent revive.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	keyOrName, message, variant string,
) (key string, multipleMatches bool, err error) {
	if !slices.Contains(xstatus.AllowedVariants, variant) {
		return "", false, ErrInvalidVariant
	}
	overlay := func(st *Status[any]) error {
		st.Message = message
		st.Variant = xstatus.Variant(variant)
		st.Time = telem.Now()
		return nil
	}
	if identifier.IsKey(keyOrName) {
		return keyOrName, false, s.WithTx(ctx, func(tx gorp.Tx) error {
			return s.NewWriter(tx).Update(ctx, keyOrName, overlay)
		})
	}
	err = s.WithTx(ctx, func(tx gorp.Tx) error {
		var ierr error
		key, multipleMatches, ierr = s.NewWriter(tx).UpsertByName(ctx, keyOrName, overlay)
		return ierr
	})
	return key, multipleMatches, err
}

// DeleteByKeyOrName deletes a status by UUID key (count 0 or 1) or by name
// (count = matches; deletes all on multi-match). Both paths run their
// retrieve + delete in a single transaction so concurrent deletes can't
// produce a stale count.
func (s *Service) DeleteByKeyOrName(ctx context.Context, keyOrName string) (int, error) {
	var count int
	err := s.WithTx(ctx, func(tx gorp.Tx) error {
		if identifier.IsKey(keyOrName) {
			var st Status[any]
			rerr := s.NewRetrieve().Where(MatchKeys[any](keyOrName)).Entry(&st).Exec(ctx, tx)
			if errors.Is(rerr, query.ErrNotFound) {
				return nil
			}
			if rerr != nil {
				return rerr
			}
			if derr := s.NewWriter(tx).Delete(ctx, keyOrName); derr != nil {
				return derr
			}
			count = 1
			return nil
		}
		var ierr error
		count, ierr = s.NewWriter(tx).DeleteByName(ctx, keyOrName)
		return ierr
	})
	return count, err
}
