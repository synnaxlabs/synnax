// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/synnaxlabs/gorp"
)

// DropDangling removes from each stored schematic the links that
// gorp.Table.DropDangling removes, such as a channel key that the Core never issued,
// and writes each schematic that it changes. A deleted channel's key stays.
func (s *Service) DropDangling(ctx context.Context, tx gorp.Tx) error {
	var stored []Schematic
	if err := s.table.NewRetrieve().Entries(&stored).Exec(ctx, tx); err != nil {
		return err
	}
	for i := range stored {
		dropped, err := s.table.DropDangling(ctx, tx, &stored[i])
		if err != nil {
			return err
		}
		if !dropped {
			continue
		}
		if err = s.table.NewCreate().Entry(&stored[i]).Exec(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
