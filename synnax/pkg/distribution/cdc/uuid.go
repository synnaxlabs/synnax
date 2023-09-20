// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cdc

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"io"
	"strings"
)

type GorpConfig[K gorp.Key, E gorp.Entry[K]] struct {
	DB       *gorp.DB
	DataType telem.DataType
	Marshal  func(K) []byte
}

func UUIDGorpConfig[E gorp.Entry[uuid.UUID]](db *gorp.DB) GorpConfig[uuid.UUID, E] {
	return GorpConfig[uuid.UUID, E]{
		DB:       db,
		DataType: telem.UUIDT,
		Marshal: func(k uuid.UUID) []byte {
			return k[:]
		},
	}
}

func OpenGorp[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	svc *Service,
	cfg GorpConfig[K, E],
) (io.Closer, error) {
	name := strings.ToLower(types.Name[E]())
	channels := []channel.Channel{
		{
			Name:        fmt.Sprintf("sy_%s_set", name),
			Leaseholder: core.Free,
			DataType:    cfg.DataType,
			Virtual:     true,
		},
		{
			Name:        fmt.Sprintf("sy_%s_delete", name),
			Leaseholder: core.Free,
			DataType:    cfg.DataType,
			Virtual:     true,
		},
	}
	obs := observe.Translator[gorp.TxReader[K, E], []change.Change[[]byte, struct{}]]{
		Observable: gorp.Observe[K, E](cfg.DB),
		Translate: func(r gorp.TxReader[K, E]) []change.Change[[]byte, struct{}] {
			changes := iter.ToSlice[change.Change[K, E]](ctx, r)
			var out []change.Change[[]byte, struct{}]
			for _, c := range changes {
				out = append(out, change.Change[[]byte, struct{}]{
					Key:     cfg.Marshal(c.Key),
					Variant: c.Variant,
				})
			}
			return out
		},
	}
	return svc.OpenCore(ctx, CoreConfig{
		Set:    channels[0],
		Delete: channels[1],
		Obs:    obs,
	})
}
