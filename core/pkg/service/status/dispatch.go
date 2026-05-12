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

// AllowedVariants enumerates the variant strings accepted by SetByKeyOrName.
var AllowedVariants = []string{
	string(xstatus.VariantSuccess),
	string(xstatus.VariantInfo),
	string(xstatus.VariantWarning),
	string(xstatus.VariantError),
	string(xstatus.VariantLoading),
	string(xstatus.VariantDisabled),
}

// ErrInvalidVariant signals a variant string outside AllowedVariants.
var ErrInvalidVariant = errors.New("invalid status variant")

// SetByKeyOrName upserts a status by UUID key or by name. On by-name multi-match,
// writes to the first by key order and returns multipleMatches=true.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	keyOrName, message, variant string,
) (key string, multipleMatches bool, err error) {
	if !slices.Contains(AllowedVariants, variant) {
		return "", false, ErrInvalidVariant
	}
	overlay := func(st *Status[any]) error {
		st.Message = message
		st.Variant = xstatus.Variant(variant)
		st.Time = telem.Now()
		return nil
	}
	if identifier.IsKey(keyOrName) {
		return keyOrName, false, s.NewWriter(nil).Update(ctx, keyOrName, overlay)
	}
	err = s.WithTx(ctx, func(tx gorp.Tx) error {
		var ierr error
		key, multipleMatches, ierr = s.NewWriter(tx).UpsertByName(ctx, keyOrName, overlay)
		return ierr
	})
	return key, multipleMatches, err
}

// DeleteByKeyOrName deletes a status by UUID key (count 0 or 1) or by name
// (count = matches; deletes all on multi-match).
func (s *Service) DeleteByKeyOrName(ctx context.Context, keyOrName string) (int, error) {
	if identifier.IsKey(keyOrName) {
		var st Status[any]
		err := s.NewRetrieve().Where(MatchKeys[any](keyOrName)).Entry(&st).Exec(ctx, nil)
		if errors.Is(err, query.ErrNotFound) {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		return 1, s.NewWriter(nil).Delete(ctx, keyOrName)
	}
	var count int
	err := s.WithTx(ctx, func(tx gorp.Tx) error {
		var ierr error
		count, ierr = s.NewWriter(tx).DeleteByName(ctx, keyOrName)
		return ierr
	})
	return count, err
}
