// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"context"
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Framer", func() {
	var (
		cluster      *mock.Cluster
		node         mock.Node
		userKey      uuid.UUID
		serviceLayer *service.Layer
		apiLayer     *api.Layer
		ctx          freighter.Context
	)
	BeforeEach(func() {
		bgCtx := context.Background()
		cluster = mock.NewCluster()
		node = cluster.Provision(bgCtx)
		securityProvider := MustSucceed(security.NewProvider())
		serviceLayer = MustSucceed(service.Open(bgCtx, service.Config{
			Distribution: node.Layer,
			Security:     securityProvider,
		}))
		userKey = uuid.New()
		Expect(serviceLayer.User.DB.WithTx(bgCtx, func(tx gorp.Tx) error {
			return serviceLayer.User.NewWriter(tx).Create(ctx, &user.User{
				Key:      userKey,
				Username: "test",
				RootUser: true,
			})
		})).To(Succeed())
		Expect(serviceLayer.RBAC.DB.WithTx(bgCtx, func(tx gorp.Tx) error {
			return serviceLayer.RBAC.NewWriter(tx).Create(ctx, &rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(userKey)},
				Objects:  []ontology.ID{rbac.AllowAllOntologyID},
				Actions:  []access.Action{},
			})
		})).To(Succeed())
		apiLayer = MustSucceed(api.New(api.Config{
			Distribution: node.Layer,
			Service:      serviceLayer,
		}))
		ctx = freighter.Context{
			Context: bgCtx,
			Params:  freighter.Params{"Subject": user.OntologyID(userKey)},
		}
	})
	AfterEach(func() {
		Expect(serviceLayer.Close()).To(Succeed())
		Expect(node.Close()).To(Succeed())
		Expect(cluster.Close()).To(Succeed())
	})
	Describe("Read", func() {
		It("should read a lot of data", func() {
			return
			res := MustSucceed(apiLayer.Channel.Create(ctx, api.ChannelCreateRequest{
				Channels: []api.Channel{
					{
						Name:     "test_index",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				},
			}))
			indexCh := res.Channels[0]
			res = MustSucceed(apiLayer.Channel.Create(ctx, api.ChannelCreateRequest{
				Channels: []api.Channel{
					{
						Name:     "test_data",
						DataType: telem.Float32T,
						Index:    indexCh.Key,
					},
				},
			}))
			dataCh := res.Channels[0]
			// write 1 GB of data to the data channel
			startingTS := telem.SecondTS
			w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{indexCh.Key, dataCh.Key},
				Start: startingTS,
			}))
			// 1 GB data - timestamps are int64 which is 8 bytes, and float32 is 4
			// bytes. So 1 GB / (12 bytes / sample) = about 80 million samples.
			// we will write a total of 100 million samples, which will be in frames of
			// size 1000, so we will write 100,000 frames.
			dataSlice := make([]float32, 1000)
			for i := range 1000 {
				dataSlice[i] = float32(i)
			}
			dataSeries := telem.NewSeriesV(dataSlice...)
			timeSlice := make([]telem.TimeStamp, 1000)
			for i := 0; i < 1_000_000; i++ {
				for i := range 1000 {
					timeSlice[i] = startingTS.Add(telem.TimeSpan(i))
				}
				startingTS = timeSlice[999] + 1
				fr := frame.NewMulti(
					[]channel.Key{indexCh.Key, dataCh.Key},
					[]telem.Series{
						telem.NewSeriesV(timeSlice...),
						dataSeries,
					},
				)
				authorized := MustSucceed(w.Write(fr))
				Expect(authorized).To(BeTrue())
			}
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			timeRG := telem.TimeRange{End: startingTS, Start: telem.SecondTS}
			readRes := MustSucceed(apiLayer.Framer.Read(ctx, api.FrameReadRequest{
				Keys:      []channel.Key{indexCh.Key, dataCh.Key},
				TimeRange: timeRG,
			}))
			MustSucceed(readRes.Read(ctx))
			i := int64(0)
			var (
				v   any
				err error
			)
			for v, err = readRes.Read(ctx); err == nil; v, err = readRes.Read(ctx) {
				fr := v.(api.Frame)
				Expect(fr.Keys()).To(ConsistOf(indexCh.Key, dataCh.Key))
				i += fr.Len()
			}
			Expect(err).To(MatchError(io.EOF))
			Expect(i).To(Equal(1_000_000_000))
		})
	})
})
