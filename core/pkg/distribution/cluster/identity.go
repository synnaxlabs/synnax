// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/version"
)

// record is the shape Aspen stores under identityKey. A Core keeps that shape so that a
// Core running Aspen can still open the data directory.
type record struct {
	Nodes      map[node.Key]recordNode
	HostKey    node.Key
	ClusterKey uuid.UUID
}

type recordNode struct {
	Address   address.Address
	Heartbeat version.Heartbeat
	State     uint32
	Key       node.Key
}

var (
	identityKey = []byte("aspen.cluster")
	// identityCodec writes JSON. Cores from v0.39 to v0.53 wrote MsgPack, and earlier
	// Cores wrote Gob.
	identityCodec = encoding.NewDecodeFallbackCodec(
		json.Codec,
		msgpack.Codec,
		gob.Codec,
	)
)

// OpenIdentity reads the identity of the Core from db. When db holds none, it creates
// one with host key 1 and a new cluster key, reachable at the address host.
func OpenIdentity(
	ctx context.Context,
	db kv.DB,
	host address.Address,
) (Identity, error) {
	b, closer, err := db.Get(ctx, identityKey)
	if errors.Is(err, query.ErrNotFound) {
		return createIdentity(ctx, db, host)
	}
	if err != nil {
		return Identity{}, err
	}
	var r record
	err = identityCodec.Decode(b, &r)
	if err = errors.Combine(err, closer.Close()); err != nil {
		return Identity{}, errors.Wrap(err, "failed to decode the identity of the Core")
	}
	return Identity{Key: r.ClusterKey, HostKey: r.HostKey}, nil
}

func createIdentity(
	ctx context.Context,
	db kv.DB,
	host address.Address,
) (Identity, error) {
	id := Identity{Key: uuid.New(), HostKey: node.KeyBootstrapper}
	b, err := identityCodec.Encode(record{
		Nodes: map[node.Key]recordNode{
			id.HostKey: {Address: host, Key: id.HostKey},
		},
		HostKey:    id.HostKey,
		ClusterKey: id.Key,
	})
	if err != nil {
		return Identity{}, err
	}
	if err = db.Set(ctx, identityKey, b); err != nil {
		return Identity{}, err
	}
	return id, nil
}
