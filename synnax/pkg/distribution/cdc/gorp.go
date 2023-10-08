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
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"io"
	"strings"
)

// GorpConfig is the configuration for opening a CDC pipeline that subscribes
// changes to a particular entry type in a gorp.DB. It's not typically necessary
// to instantiate this configuration directly, instead use a helper function
// such as GorpConfigUUID.
type GorpConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// DB is the DB to subscribe to.
	DB *gorp.DB
	// DataType is the data type of the key used by the DB.
	DataType telem.DataType
	Marshal  func(K) []byte
}

// GorpConfigUUID is a helper function for creating a CDC pipeline that propagates
// changes to UUID keyed gorp entries written to the provided DB. The returned
// configuration should be passed to SubscribeToGorp.
func GorpConfigUUID[E gorp.Entry[uuid.UUID]](db *gorp.DB) GorpConfig[uuid.UUID, E] {
	return GorpConfig[uuid.UUID, E]{
		DB:       db,
		DataType: telem.UUIDT,
		Marshal:  func(k uuid.UUID) []byte { return k[:] },
	}
}

// SubscribeToGorp opens a CDC pipeline that subscribes to the sets and deletes of a
// particular entry type in the configured gorp.DB. The returned io.Closer should be
// closed to stop the CDC pipeline.
func SubscribeToGorp[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	svc *Provider,
	cfg GorpConfig[K, E],
) (io.Closer, error) {
	var (
		name = strings.ToLower(types.Name[E]())
		obs  = observe.Translator[gorp.TxReader[K, E], []change.Change[[]byte, struct{}]]{
			Observable: gorp.Observe[K, E](cfg.DB),
			Translate: func(r gorp.TxReader[K, E]) []change.Change[[]byte, struct{}] {
				out := make([]change.Change[[]byte, struct{}], 0, r.Count())
				for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
					out = append(out, change.Change[[]byte, struct{}]{
						Key:     cfg.Marshal(c.Key),
						Variant: c.Variant,
					})
				}
				return out
			},
		}
		obsCfg = ObservableConfig{
			Name:       name,
			Observable: obs,
			Set: channel.Channel{
				Name:     fmt.Sprintf("sy_%s_set", name),
				DataType: cfg.DataType,
			},
			Delete: channel.Channel{
				Name:     fmt.Sprintf("sy_%s_delete", name),
				DataType: cfg.DataType,
			},
		}
	)
	return svc.SubscribeToObservable(ctx, obsCfg)
}
