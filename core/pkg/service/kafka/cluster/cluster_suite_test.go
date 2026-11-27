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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var db *gorp.DB

// secretsKey encrypts the secrets of every spec, which share their database.
var secretsKey = secret.NewEncryptionKey()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
	createChannels(ctx)
})

// createChannels stores the channels with keys 1 through 9 that the specs bind, since a
// cluster refuses a read of a channel that does not exist.
func createChannels(ctx SpecContext) {
	GinkgoHelper()
	table := MustOpen(
		gorp.OpenTable(ctx, gorp.TableConfig[channel.Key, channel.Channel]{
			DB:   db,
			Name: "channel",
		}),
	)
	for k := range channel.LocalKey(9) {
		Expect(table.NewCreate().
			Entry(&channel.Channel{LocalKey: k + 1, Name: "ch"}).
			Exec(ctx, db)).To(Succeed())
	}
}

func TestCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Kafka Cluster Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
