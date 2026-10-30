// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
)

// ResolveReadConfig addresses each unresolved channel in cfg to the input entry its
// slave lists under the channel's entry name, with no scale. A channel stays unresolved
// when the name matches no entry, or entries at more than one address.
func ResolveReadConfig(
	ctx context.Context,
	tx gorp.Tx,
	slaves v2.Slaves,
	cfg *ReadConfig,
) error {
	found, err := v2.ReadSlaves(
		ctx,
		tx,
		slaves,
		lo.FilterMap(cfg.Channels, func(ch ReadChannel, _ int) (string, bool) {
			u, ok := ch.Variant.(UnresolvedReadChannel)
			return u.Device, ok
		}),
	)
	if err != nil {
		return err
	}
	for i, ch := range cfg.Channels {
		u, ok := ch.Variant.(UnresolvedReadChannel)
		if !ok {
			continue
		}
		if e, ok := v2.FindEntry(found[u.Device].Inputs, u.Pdo); ok {
			cfg.Channels[i].Variant = EntryReadChannel{
				BaseReadChannel: u.BaseReadChannel,
				Index:           e.Index,
				SubIndex:        e.SubIndex,
				DataType:        e.DataType,
				Scale:           Scale{Variant: NoneScale{}},
			}
		}
	}
	return nil
}
