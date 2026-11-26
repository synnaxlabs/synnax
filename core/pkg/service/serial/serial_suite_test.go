// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package serial_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	node       testutil.Node
	db         *gorp.DB
	statusSvc  *status.Service
	channelSvc *channel.Service
	icdSvc     *icd.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	icdSvc = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
		DB:   db,
		ImEx: imex.NewService(),
	}))
})

// createChannel creates a channel of data type dt, indexed by idx, or an index channel
// when idx is the zero channel.
func createChannel(
	ctx context.Context,
	dt telem.DataType,
	idx channel.Channel,
) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:        UniqueChannelName(),
		DataType:    dt,
		Leaseholder: node.Identity.HostKey,
		IsIndex:     idx.LocalKey == 0,
		LocalIndex:  idx.LocalKey,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}

func TestSerial(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Serial Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
