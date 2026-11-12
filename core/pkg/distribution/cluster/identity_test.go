// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/version"
)

var identityKey = []byte("aspen.cluster")

const clusterKey = "5f0c8a52-3b8e-4d0e-9a6c-1e7f2b9d4c31"

// oldRecord and oldNode have the field names of the identity that Cores before v0.54
// wrote in MsgPack or Gob.
type (
	oldRecord struct {
		Nodes      map[node.Key]oldNode
		HostKey    node.Key
		ClusterKey uuid.UUID
	}
	oldNode struct {
		Address   address.Address
		Heartbeat version.Heartbeat
		State     uint32
		Key       node.Key
	}
)

func aspenRecord(hostKey int, nodeKeys ...int) []byte {
	nodes := ""
	for i, k := range nodeKeys {
		if i > 0 {
			nodes += ","
		}
		nodes += fmt.Sprintf(
			`"%d":{"Address":"localhost:%d","Heartbeat":{"Generation":3,"Version":7},"State":0,"Key":%d}`,
			k,
			9089+k,
			k,
		)
	}
	return fmt.Appendf(
		nil,
		`{"Nodes":{%s},"HostKey":%d,"ClusterKey":%q}`,
		nodes, hostKey, clusterKey,
	)
}

var _ = Describe("Identity", func() {
	var db kv.DB
	BeforeEach(func() { db = DeferClose(memkv.New()) })

	It(
		"Should create an identity with host key 1 when the store has none",
		func(ctx SpecContext) {
			id := MustSucceed(cluster.OpenIdentity(ctx, db, "localhost:9090"))
			Expect(id.HostKey).To(Equal(node.KeyBootstrapper))
			Expect(id.Key).ToNot(Equal(uuid.Nil()))
			Expect(cluster.OpenIdentity(ctx, db, "localhost:9091")).To(Equal(id))
		},
	)

	It("Should store a new identity that an older Core reads", func(ctx SpecContext) {
		id := MustSucceed(cluster.OpenIdentity(ctx, db, "localhost:9090"))
		b, closer := MustSucceed2(db.Get(ctx, identityKey))
		defer func() { Expect(closer.Close()).To(Succeed()) }()
		Expect(b).To(MatchJSON(fmt.Sprintf(
			`{"Nodes":{"1":{"Address":"localhost:9090","Heartbeat":{"Generation":0,"Version":0},"State":0,"Key":1}},"HostKey":1,"ClusterKey":%q}`,
			id.Key.String(),
		)))
	})

	It("Should read the identity an older Core stored", func(ctx SpecContext) {
		Expect(db.Set(ctx, identityKey, aspenRecord(1, 1))).To(Succeed())
		Expect(
			cluster.OpenIdentity(ctx, db, "localhost:9090"),
		).To(Equal(cluster.Identity{
			Key:     MustSucceed(uuid.Parse(clusterKey)),
			HostKey: 1,
		}))
	})

	DescribeTable(
		"Should read the identity an older Core encoded",
		func(ctx SpecContext, codec encoding.Codec) {
			b := MustSucceed(codec.Encode(oldRecord{
				Nodes: map[node.Key]oldNode{
					1: {
						Address:   "localhost:9090",
						Heartbeat: version.Heartbeat{Generation: 3, Version: 7},
						Key:       1,
					},
				},
				HostKey:    1,
				ClusterKey: MustSucceed(uuid.Parse(clusterKey)),
			}))
			Expect(db.Set(ctx, identityKey, b)).To(Succeed())
			Expect(
				cluster.OpenIdentity(ctx, db, "localhost:9090"),
			).To(Equal(cluster.Identity{
				Key:     MustSucceed(uuid.Parse(clusterKey)),
				HostKey: 1,
			}))
		},
		Entry("in MsgPack", msgpack.Codec),
		Entry("in Gob", gob.Codec),
	)

	It(
		"Should return an error when the stored identity does not decode",
		func(ctx SpecContext) {
			Expect(db.Set(ctx, identityKey, []byte("not an identity"))).To(Succeed())
			Expect(cluster.OpenIdentity(ctx, db, "localhost:9090")).Error().
				To(MatchError(ContainSubstring(
					"failed to decode the identity of the Core",
				)))
		},
	)
})
