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
// by key, reading through tx. It returns query.ErrNotFound when no group has key.
//
// Callers that go on to write must pass the same tx they write under, so the IDs they
// act on and the IDs the write touches come from one snapshot.
func (s *Service) RetrieveGroupSymbols(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
) ([]resource.ID, error) {
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Entry(&group.Group{}).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	var symbols []Symbol
	if err := s.NewRetrieve().
		Where(MatchGroups(key)).
		Entries(&symbols).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(symbols, func(sym Symbol, _ int) resource.ID {
		return sym.ResourceID()
	}), nil
}

// DeleteGroup deletes the symbols that ids names, then the group identified by key.
// Pass the ids RetrieveGroupSymbols returned under tx, so the symbols removed and the
// symbols the caller enforced access on come from one snapshot. It returns a validation
// error when the group still holds anything after the symbols are deleted.
func (s *Service) DeleteGroup(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
	ids []resource.ID,
) error {
	keys, err := KeysFromResourceIDs(ids)
	if err != nil {
		return err
	}
	if err = s.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return err
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, key)
}
