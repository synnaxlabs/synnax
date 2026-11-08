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

// record is the shape Aspen stores under identityKey.
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
	// identityCodec reads JSON. Cores from v0.39 to v0.53 wrote MsgPack, and earlier
	// Cores wrote Gob.
	identityCodec = encoding.NewDecodeFallbackCodec(
		json.Codec,
		msgpack.Codec,
		gob.Codec,
	)
)

// CheckIdentity returns an error when db holds the identity of a node other than 1, or
// of a cluster that lists other nodes.
func CheckIdentity(ctx context.Context, db kv.DB) error {
	b, closer, err := db.Get(ctx, identityKey)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var r record
	err = identityCodec.Decode(ctx, b, &r)
	if err = errors.Combine(err, closer.Close()); err != nil {
		return errors.Wrap(err, "failed to decode the identity of the Core")
	}
	if r.HostKey != node.KeyBootstrapper || len(r.Nodes) > 1 {
		return errors.Newf(
			"the data directory belongs to node %d of a cluster, and a Core no longer runs as a cluster",
			r.HostKey,
		)
	}
	return nil
}
