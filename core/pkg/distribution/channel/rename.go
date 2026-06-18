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

// Rename renames the storage channels for the provided keys, routing each key to its
// leaseholder. Free-virtual channels have no persistent storage and are skipped.
func (s *Service) Rename(ctx context.Context, keys Keys, names []string) error {
	batch := s.renameRouter.Batch(newRenameBatch(keys, names))
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		if err := s.renameRemote(ctx, nodeKey, keys, names); err != nil {
			return err
		}
	}
	if len(batch.Gateway) == 0 {
		return nil
	}
	keys, names = unzipRenameBatch(batch.Gateway)
	return s.cfg.TSDB.RenameChannels(ctx, keys.Storage(), names)
}

func (s *Service) renameHandler(ctx context.Context, msg RenameRequest) (types.Nil, error) {
	return types.Nil{}, s.cfg.TSDB.RenameChannels(ctx, msg.Keys.Storage(), msg.Names)
}

func (s *Service) renameRemote(ctx context.Context, target node.Key, keys Keys, names []string) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = s.cfg.Transport.RenameClient().Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
	return err
}

type renameBatchEntry struct {
	name string
	key  Key
}

var _ proxy.Entry = renameBatchEntry{}

func (r renameBatchEntry) Lease() node.Key { return r.key.Lease() }

func unzipRenameBatch(entries []renameBatchEntry) (Keys, []string) {
	return lo.UnzipBy2(entries, func(e renameBatchEntry) (Key, string) {
		return e.key, e.name
	})
}

func newRenameBatch(keys Keys, names []string) []renameBatchEntry {
	return lo.ZipBy2(keys, names, func(k Key, n string) renameBatchEntry {
		return renameBatchEntry{key: k, name: n}
	})
}
