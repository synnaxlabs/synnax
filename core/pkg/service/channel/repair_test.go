// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Data type repair", func() {
	It("Should derive the data type of a stored calculated channel and log the rest",
		func(ctx SpecContext) {
			dist := testutil.NewNode(ctx)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[channel.Key, channel.Channel]{
				DB: dist.DB,
			}))
			calculated := channel.Channel{
				Name:        "calc",
				Leaseholder: node.KeyFree,
				LocalKey:    7001,
				Virtual:     true,
				Expression:  "return 1",
			}
			free := channel.Channel{
				Name:        "free",
				Leaseholder: node.KeyFree,
				LocalKey:    7002,
				Virtual:     true,
			}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entries(&[]channel.Channel{calculated, free}).
				Exec(ctx, dist.DB)).To(Succeed())
			core, logs := observer.New(zapcore.WarnLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			svc := openService(ctx, dist, channel.ServiceConfig{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			})
			var stored []channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(calculated.Key(), free.Key())).
				Entries(&stored).
				Exec(ctx, dist.DB)).To(Succeed())
			byName := make(map[string]telem.DataType, len(stored))
			for _, ch := range stored {
				byName[ch.Name] = ch.DataType
			}
			Expect(byName).To(Equal(map[string]telem.DataType{
				"calc": telem.Int64T,
				"free": telem.UnknownT,
			}))
			warned := logs.FilterMessage(
				"channel has no data type, so the Core refuses its next update",
			).All()
			Expect(warned).To(HaveLen(1))
			Expect(warned[0].ContextMap()).To(HaveKeyWithValue("name", "free"))
			Expect(warned[0].ContextMap()).
				To(HaveKeyWithValue("key", free.Key().String()))
		},
	)
})
