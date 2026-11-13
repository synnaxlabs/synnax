// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	channelSvc *channel.Service
	framerSvc  *framer.Service
)

func TestIngest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sift Ingest Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
})

// createIndex creates an index channel with a unique name.
func createIndex(ctx context.Context) channel.Channel {
	GinkgoHelper()
	idx := channel.Channel{
		Name:     "time_" + uuid.New().String()[:8],
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &idx)
	})).To(Succeed())
	return idx
}

// createData creates a data channel of dt on idx with a unique name.
func createData(
	ctx context.Context,
	idx channel.Channel,
	dt telem.DataType,
) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:       "data_" + uuid.New().String()[:8],
		DataType:   dt,
		LocalIndex: idx.LocalKey,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}

// write writes one frame of the index and data series to storage.
func write(
	ctx context.Context,
	idx channel.Channel,
	stamps []telem.TimeStamp,
	data map[channel.Key]telem.Series,
) {
	GinkgoHelper()
	keys := channel.Keys{idx.Key()}
	series := []telem.Series{telem.NewSeries(stamps)}
	for key, s := range data {
		keys = append(keys, key)
		series = append(series, s)
	}
	w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: stamps[0],
	}))
	Expect(MustSucceed(w.Write(frame.NewMulti(keys, series)))).To(BeTrue())
	Expect(w.Close()).To(Succeed())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
