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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// ErrInvalidVariant signals a variant string rejected by xstatus.IsVariant.
var ErrInvalidVariant = errors.New("invalid status variant")

// ErrEmptyKeyOrName signals a missing key_or_name parameter.
var ErrEmptyKeyOrName = errors.New("key_or_name is required")

// SetByKeyOrName tries to update a row whose Key equals keyOrName, then a row
// whose Name matches, and finally creates a new row with Key=Name=keyOrName.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	keyOrName, message, variant string,
) (key string, multipleMatches bool, err error) {
	if keyOrName == "" {
		return "", false, ErrEmptyKeyOrName
	}
	if !xstatus.IsVariant(variant) {
		return "", false, ErrInvalidVariant
	}
	overlay := func(st *Status[any]) error {
		st.Message = message
		st.Variant = xstatus.Variant(variant)
		st.Time = telem.Now()
		return nil
	}
	err = s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.NewWriter(tx)
		if uerr := w.Update(ctx, keyOrName, overlay); uerr == nil {
			key = keyOrName
			return nil
		} else if !errors.Is(uerr, query.ErrNotFound) {
			return uerr
		}
		matches, merr := w.retrieveByName(ctx, keyOrName)
		if merr != nil {
			return merr
		}
		if len(matches) > 0 {
			matched := matches[0]
			multipleMatches = len(matches) > 1
			if oerr := overlay(&matched); oerr != nil {
				return oerr
			}
			key = matched.Key
			return w.Set(ctx, &matched)
		}
		fresh := Status[any]{Key: keyOrName, Name: keyOrName, Variant: xstatus.VariantInfo}
		if oerr := overlay(&fresh); oerr != nil {
			return oerr
		}
		key = keyOrName
		return w.Set(ctx, &fresh)
	})
	return key, multipleMatches, err
}

// DeleteByKeyOrName deletes a row whose Key equals keyOrName (count 0 or 1) or,
// failing that, all rows whose Name matches (count = matches).
func (s *Service) DeleteByKeyOrName(ctx context.Context, keyOrName string) (int, error) {
	if keyOrName == "" {
		return 0, ErrEmptyKeyOrName
	}
	var count int
	err := s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.NewWriter(tx)
		var st Status[any]
		rerr := s.NewRetrieve().Where(MatchKeys[any](keyOrName)).Entry(&st).Exec(ctx, tx)
		if rerr == nil {
			if derr := w.Delete(ctx, keyOrName); derr != nil {
				return derr
			}
			count = 1
			return nil
		}
		if !errors.Is(rerr, query.ErrNotFound) {
			return rerr
		}
		var ierr error
		count, ierr = w.DeleteByName(ctx, keyOrName)
		return ierr
	})
	return count, err
}
