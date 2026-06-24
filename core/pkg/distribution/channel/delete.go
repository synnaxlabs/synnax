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
	"go/types"
)

// Delete deletes the storage channels for the provided keys, routing each key to its
// leaseholder. Free-virtual channels have no persistent storage and are skipped. It
// does not touch channel metadata.
func (s *Service) Delete(ctx context.Context, keys Keys) error {
	batch := s.deleteRouter.Batch(keys)
	for nodeKey, entries := range batch.Peers {
		addr, err := s.cfg.HostResolver.Resolve(nodeKey)
		if err != nil {
			return err
		}
		if _, err := s.cfg.Transport.DeleteClient().Send(
			ctx, addr, DeleteRequest{Keys: entries},
		); err != nil {
			return err
		}
	}
	return s.cfg.TS.DeleteChannels(Keys(batch.Gateway).Storage())
}

func (s *Service) deleteHandler(
	ctx context.Context, req DeleteRequest,
) (types.Nil, error) {
	return types.Nil{}, s.cfg.TS.DeleteChannels(req.Keys.Storage())
}
