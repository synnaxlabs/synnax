// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
)

// DeleteSymbolsIn deletes every symbol in the group identified by key, and leaves the
// group. It returns query.ErrNotFound when no group has key.
func (s *Service) DeleteSymbolsIn(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
) error {
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Entry(&group.Group{}).
		Exec(ctx, tx); err != nil {
		return err
	}
	var symbols []Symbol
	if err := s.NewRetrieve().
		Where(MatchGroups(key)).
		Entries(&symbols).
		Exec(ctx, tx); err != nil {
		return err
	}
	keys := lo.Map(symbols, func(sym Symbol, _ int) Key { return sym.Key })
	return s.NewWriter(tx).Delete(ctx, keys...)
}
