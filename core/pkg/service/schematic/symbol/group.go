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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// RetrieveGroupSymbols returns the resource ID of every symbol in the group identified
// by key, reading through snap. It returns query.ErrNotFound when no group has key.
func (s *Service) RetrieveGroupSymbols(
	ctx context.Context,
	snap gorp.Snapshot,
	key group.Key,
) ([]resource.ID, error) {
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Entry(&group.Group{}).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	var symbols []Symbol
	if err := s.NewRetrieve().
		Where(MatchGroups(key)).
		Entries(&symbols).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.Map(symbols, func(sym Symbol, _ int) resource.ID {
		return sym.ResourceID()
	}), nil
}

// DeleteGroup deletes the group identified by key and every symbol in it. It returns
// query.ErrNotFound when no group has key, and a validation error when the group holds
// anything other than symbols.
func (s *Service) DeleteGroup(ctx context.Context, tx gorp.Tx, key group.Key) error {
	ids, err := s.RetrieveGroupSymbols(ctx, tx, key)
	if err != nil {
		return err
	}
	keys, err := KeysFromResourceIDs(ids)
	if err != nil {
		return err
	}
	if err = s.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return err
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, key)
}
