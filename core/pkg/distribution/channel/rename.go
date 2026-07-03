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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/unsafe"
)

type renameBatchEntry struct {
	name string
	key  Key
}

var _ proxy.Entry = renameBatchEntry{}

func (r renameBatchEntry) Lease() node.Key { return r.key.Lease() }

// Rename renames each storage channel keyed in renames to its corresponding new name,
// routing each key to its leaseholder. Free-virtual channels have no persistent storage
// and are skipped.
func (s *Service) Rename(ctx context.Context, renames map[Key]string) error {
	entries := lo.MapToSlice(renames, func(key Key, name string) renameBatchEntry {
		return renameBatchEntry{key: key, name: name}
	})
	batch := s.renameRouter.Batch(entries)
	for nodeKey, entries := range batch.Peers {
		addr, err := s.cfg.HostResolver.Resolve(nodeKey)
		if err != nil {
			return err
		}
		_, err = s.cfg.Transport.RenameClient().
			Send(ctx, addr, RenameRequest{Renames: zipRenameBatch(entries)})
		if err != nil {
			return err
		}
	}
	return s.cfg.TS.RenameChannels(ctx, storageRenames(zipRenameBatch(batch.Gateway)))
}

func (s *Service) renameHandler(
	ctx context.Context, req RenameRequest,
) (types.Nil, error) {
	return types.Nil{}, s.cfg.TS.RenameChannels(ctx, storageRenames(req.Renames))
}

func zipRenameBatch(entries []renameBatchEntry) map[Key]string {
	return lo.SliceToMap(entries, func(e renameBatchEntry) (Key, string) {
		return e.key, e.name
	})
}

func storageRenames(renames map[Key]string) map[ts.ChannelKey]string {
	return unsafe.ReinterpretMapKeys[Key, ts.ChannelKey](renames)
}
