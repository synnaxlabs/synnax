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
	entries := make([]renameBatchEntry, 0, len(renames))
	for key, name := range renames {
		entries = append(entries, renameBatchEntry{key: key, name: name})
	}
	batch := s.renameRouter.Batch(entries)
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		addr, err := s.cfg.HostResolver.Resolve(nodeKey)
		if err != nil {
			return err
		}
		_, err = s.cfg.Transport.RenameClient().
			Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
		if err != nil {
			return err
		}
	}
	keys, names := unzipRenameBatch(batch.Gateway)
	return s.cfg.TS.RenameChannels(ctx, keys.Storage(), names)
}

func (s *Service) renameHandler(
	ctx context.Context, req RenameRequest,
) (types.Nil, error) {
	return types.Nil{}, s.cfg.TS.RenameChannels(ctx, req.Keys.Storage(), req.Names)
}

func unzipRenameBatch(entries []renameBatchEntry) (Keys, []string) {
	return lo.UnzipBy2(entries, func(e renameBatchEntry) (Key, string) {
		return e.key, e.name
	})
}
