// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fake_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	assetsv1 "github.com/sift-stack/sift/go/gen/sift/assets/v1"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv2 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v2"
	pingv1 "github.com/sift-stack/sift/go/gen/sift/ping/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Server", func() {
	var (
		server  *fake.Server
		ingest  ingestv1.IngestServiceClient
		configs ingestionconfigsv2.IngestionConfigServiceClient
		pinger  pingv1.PingServiceClient
		assets  assetsv1.AssetServiceClient
		runs    runsv2.RunServiceClient
	)

	open := func(cfg fake.Config) {
		GinkgoHelper()
		server = MustOpen(fake.Open(cfg))
		conn := MustOpen(grpc.NewClient(
			server.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		))
		ingest = ingestv1.NewIngestServiceClient(conn)
		configs = ingestionconfigsv2.NewIngestionConfigServiceClient(conn)
		pinger = pingv1.NewPingServiceClient(conn)
		assets = assetsv1.NewAssetServiceClient(conn)
		runs = runsv2.NewRunServiceClient(conn)
	}

	createConfig := func(ctx context.Context, clientKey string) string {
		GinkgoHelper()
		res := MustSucceed(configs.CreateIngestionConfig(
			ctx,
			&ingestionconfigsv2.CreateIngestionConfigRequest{
				AssetName: "stand",
				ClientKey: clientKey,
				Flows: []*ingestionconfigsv2.FlowConfig{{
					Name: "time",
					Channels: []*ingestionconfigsv2.ChannelConfig{{
						Name:     "pressure",
						DataType: typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE,
					}},
				}},
			},
		))
		return res.IngestionConfig.IngestionConfigId
	}

	at := time.Unix(10, 0).UTC()

	request := func(
		configID string,
		t time.Time,
		v *ingestv1.IngestWithConfigDataChannelValue,
	) *ingestv1.IngestWithConfigDataStreamRequest {
		return &ingestv1.IngestWithConfigDataStreamRequest{
			IngestionConfigId: configID,
			Flow:              "time",
			Timestamp:         timestamppb.New(t),
			ChannelValues:     []*ingestv1.IngestWithConfigDataChannelValue{v},
		}
	}

	double := func(v float64) *ingestv1.IngestWithConfigDataChannelValue {
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Double{Double: v},
		}
	}

	sendAll := func(
		ctx context.Context,
		reqs ...*ingestv1.IngestWithConfigDataStreamRequest,
	) error {
		GinkgoHelper()
		stream := MustSucceed(ingest.IngestWithConfigDataStream(ctx))
		for _, req := range reqs {
			Expect(stream.Send(req)).To(Succeed())
		}
		_, err := stream.CloseAndRecv()
		return err
	}

	Describe("Address", func() {
		It("Should return a local address", func() {
			open(fake.Config{})
			Expect(server.Address()).To(HavePrefix("127.0.0.1:"))
		})
	})

	Describe("IngestWithConfigDataStream", func() {
		BeforeEach(func() { open(fake.Config{}) })

		It("Should store the values of a stream once the client confirms it",
			func(ctx SpecContext) {
				id := createConfig(ctx, "")
				stream := MustSucceed(ingest.IngestWithConfigDataStream(ctx))
				Expect(stream.Send(request(id, at, double(1.5)))).To(Succeed())
				Expect(server.Points("stand", "pressure")).To(BeEmpty())
				MustSucceed(stream.CloseAndRecv())
				Expect(server.Points("stand", "pressure")).To(Equal([]fake.Point{
					{Time: at, Value: 1.5},
				}))
				Expect(server.Confirmed()).To(Equal(1))
			})

		It("Should keep the last value sent at a timestamp", func(ctx SpecContext) {
			id := createConfig(ctx, "")
			Expect(sendAll(
				ctx,
				request(id, at, double(1.5)),
				request(id, at, double(2.5)),
				request(id, at.Add(time.Second), double(3.5)),
			)).To(Succeed())
			Expect(server.Points("stand", "pressure")).To(Equal([]fake.Point{
				{Time: at, Value: 2.5},
				{Time: at.Add(time.Second), Value: 3.5},
			}))
		})

		It("Should report a request that does not match its flow as invalid",
			func(ctx SpecContext) {
				id := createConfig(ctx, "")
				wrong := request(id, at, &ingestv1.IngestWithConfigDataChannelValue{
					Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: 1},
				})
				Expect(sendAll(ctx, wrong)).To(Succeed())
				Expect(server.Invalid()).To(HaveLen(1))
				Expect(server.Points("stand", "pressure")).To(BeEmpty())
			})

		It("Should store nothing for an empty value", func(ctx SpecContext) {
			id := createConfig(ctx, "")
			Expect(
				sendAll(ctx, request(id, at, &ingestv1.IngestWithConfigDataChannelValue{
					Type: &ingestv1.IngestWithConfigDataChannelValue_Empty{
						Empty: &emptypb.Empty{},
					},
				})),
			).To(Succeed())
			Expect(server.Invalid()).To(BeEmpty())
			Expect(server.Points("stand", "pressure")).To(BeEmpty())
		})
	})

	Describe("Fail", func() {
		It("Should end the next streams with the given errors and store nothing",
			func(ctx SpecContext) {
				open(fake.Config{})
				id := createConfig(ctx, "")
				server.Fail(status.Error(codes.Unavailable, "down"))
				Expect(sendAll(ctx, request(id, at, double(1.5)))).
					To(MatchError(ContainSubstring("down")))
				Expect(server.Points("stand", "pressure")).To(BeEmpty())
				Expect(sendAll(ctx, request(id, at, double(1.5)))).To(Succeed())
				Expect(server.Points("stand", "pressure")).To(HaveLen(1))
			})
	})

	Describe("Received", func() {
		It("Should count requests before the client confirms them",
			func(ctx SpecContext) {
				open(fake.Config{})
				id := createConfig(ctx, "")
				stream := MustSucceed(ingest.IngestWithConfigDataStream(ctx))
				Expect(stream.Send(request(id, at, double(1.5)))).To(Succeed())
				Eventually(server.Received).Should(Equal(1))
				Expect(server.Confirmed()).To(BeZero())
				MustSucceed(stream.CloseAndRecv())
			})
	})

	Describe("Streams", func() {
		It("Should count each stream a client opens", func(ctx SpecContext) {
			open(fake.Config{})
			id := createConfig(ctx, "")
			stream := MustSucceed(ingest.IngestWithConfigDataStream(ctx))
			Expect(stream.Send(request(id, at, double(1.5)))).To(Succeed())
			Eventually(server.Streams).Should(Equal(1))
			MustSucceed(stream.CloseAndRecv())
		})
	})

	Describe("Pause", func() {
		It("Should hold streams until Resume", func(ctx SpecContext) {
			open(fake.Config{})
			id := createConfig(ctx, "")
			server.Pause()
			done := make(chan error, 1)
			go func() { done <- sendAll(ctx, request(id, at, double(1.5))) }()
			Consistently(done, 50*time.Millisecond).ShouldNot(Receive())
			server.Resume()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	Describe("CreateIngestionConfig", func() {
		BeforeEach(func() { open(fake.Config{}) })

		It("Should create a config on the asset", func(ctx SpecContext) {
			createConfig(ctx, "key")
			Expect(server.IngestionConfigs()).To(HaveExactElements(
				HaveField("ClientKey", "key"),
			))
		})

		It("Should reject a client key that exists", func(ctx SpecContext) {
			createConfig(ctx, "key")
			Expect(configs.CreateIngestionConfig(
				ctx,
				&ingestionconfigsv2.CreateIngestionConfigRequest{
					AssetName: "stand",
					ClientKey: "key",
				},
			)).Error().To(MatchError(ContainSubstring("already exists")))
		})

		It("Should reject two flows with one name", func(ctx SpecContext) {
			Expect(configs.CreateIngestionConfig(
				ctx,
				&ingestionconfigsv2.CreateIngestionConfigRequest{
					AssetName: "stand",
					Flows: []*ingestionconfigsv2.FlowConfig{
						{Name: "time"},
						{Name: "time"},
					},
				},
			)).Error().To(MatchError(ContainSubstring("duplicate flow")))
		})

		It("Should require an asset name", func(ctx SpecContext) {
			Expect(configs.CreateIngestionConfig(
				ctx,
				&ingestionconfigsv2.CreateIngestionConfigRequest{},
			)).Error().To(MatchError(ContainSubstring("asset_name is required")))
		})
	})

	Describe("ListIngestionConfigs", func() {
		BeforeEach(func() { open(fake.Config{}) })

		It("Should filter by client key", func(ctx SpecContext) {
			createConfig(ctx, "a")
			b := createConfig(ctx, "b")
			res := MustSucceed(configs.ListIngestionConfigs(
				ctx,
				&ingestionconfigsv2.ListIngestionConfigsRequest{
					Filter: `client_key == "b"`,
				},
			))
			Expect(res.IngestionConfigs).To(HaveExactElements(
				HaveField("IngestionConfigId", b),
			))
		})

		It("Should list every config without a filter", func(ctx SpecContext) {
			createConfig(ctx, "a")
			createConfig(ctx, "b")
			res := MustSucceed(configs.ListIngestionConfigs(
				ctx,
				&ingestionconfigsv2.ListIngestionConfigsRequest{},
			))
			Expect(res.IngestionConfigs).To(HaveLen(2))
		})

		It("Should reject a filter it does not understand", func(ctx SpecContext) {
			Expect(configs.ListIngestionConfigs(
				ctx,
				&ingestionconfigsv2.ListIngestionConfigsRequest{
					Filter: `asset_id == "x"`,
				},
			)).Error().To(MatchError(ContainSubstring("bad filter")))
		})
	})

	Describe("Ping", func() {
		It("Should answer", func(ctx SpecContext) {
			open(fake.Config{})
			Expect(MustSucceed(pinger.Ping(ctx, &pingv1.PingRequest{})).Response).
				To(Equal("pong"))
		})
	})

	Describe("ListAssets", func() {
		It("Should list each asset once, by name, in pages", func(ctx SpecContext) {
			open(fake.Config{Assets: []string{"pad", "stand"}})
			createConfig(ctx, "")
			first := MustSucceed(assets.ListAssets(
				ctx,
				&assetsv1.ListAssetsRequest{PageSize: 1},
			))
			Expect(first.Assets).To(HaveExactElements(HaveField("Name", "pad")))
			second := MustSucceed(assets.ListAssets(
				ctx,
				&assetsv1.ListAssetsRequest{
					PageSize:  1,
					PageToken: first.NextPageToken,
				},
			))
			Expect(second.Assets).To(HaveExactElements(SatisfyAll(
				HaveField("Name", "stand"),
				HaveField("AssetId", "asset-stand"),
			)))
			Expect(second.NextPageToken).To(BeEmpty())
		})

		DescribeTable("Should reject a page it cannot list",
			func(ctx SpecContext, req *assetsv1.ListAssetsRequest, msg string) {
				open(fake.Config{Assets: []string{"pad"}})
				Expect(assets.ListAssets(ctx, req)).Error().
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("a token it did not make",
				&assetsv1.ListAssetsRequest{PageToken: "x"}, "bad page token"),
			Entry("a negative token",
				&assetsv1.ListAssetsRequest{PageToken: "-1"}, "bad page token"),
			Entry("a token past the last asset",
				&assetsv1.ListAssetsRequest{PageToken: "2"}, "bad page token"),
		)
	})

	Describe("Runs", func() {
		var (
			start = timestamppb.New(at)
			stop  = timestamppb.New(at.Add(time.Minute))
			key   = "range-1"
		)

		BeforeEach(func() { open(fake.Config{Assets: []string{"stand"}}) })

		createRun := func(
			ctx context.Context,
			req *runsv2.CreateAdhocRunRequest,
		) (*runsv2.CreateAdhocRunResponse, error) {
			if req.Name == "" {
				req.Name = "test 1"
			}
			if req.StartTime == nil {
				req.StartTime, req.StopTime = start, stop
			}
			return runs.CreateAdhocRun(ctx, req)
		}

		It("Should create a run and find it by client key", func(ctx SpecContext) {
			created := MustSucceed(createRun(ctx, &runsv2.CreateAdhocRunRequest{
				AssetIds:  []string{"asset-stand"},
				Tags:      []string{"hot"},
				ClientKey: &key,
			})).Run
			Expect(server.Runs()).To(HaveExactElements(SatisfyAll(
				HaveField("RunId", created.RunId),
				HaveField("AssetIds", ConsistOf("asset-stand")),
			)))
			res := MustSucceed(runs.ListRuns(
				ctx,
				&runsv2.ListRunsRequest{Filter: `client_key == "range-1"`},
			))
			Expect(res.Runs).To(HaveExactElements(SatisfyAll(
				HaveField("RunId", created.RunId),
				HaveField("Name", "test 1"),
				HaveField("Tags", ConsistOf("hot")),
			)))
			Expect(runs.ListRuns(
				ctx,
				&runsv2.ListRunsRequest{Filter: `client_key == "other"`},
			)).To(HaveField("Runs", BeEmpty()))
		})

		It("Should reject a second run with the same client key",
			func(ctx SpecContext) {
				MustSucceed(createRun(ctx, &runsv2.CreateAdhocRunRequest{
					ClientKey: &key,
				}))
				Expect(createRun(ctx, &runsv2.CreateAdhocRunRequest{
					ClientKey: &key,
				})).Error().To(WithTransform(status.Code, Equal(codes.AlreadyExists)))
			})

		It("Should reject an asset that does not exist", func(ctx SpecContext) {
			Expect(createRun(ctx, &runsv2.CreateAdhocRunRequest{
				AssetIds: []string{"asset-pad"},
			})).Error().To(WithTransform(status.Code, Equal(codes.NotFound)))
		})

		It("Should reject a run that stops before it starts", func(ctx SpecContext) {
			Expect(createRun(ctx, &runsv2.CreateAdhocRunRequest{
				StartTime: stop,
				StopTime:  start,
			})).Error().To(MatchError(ContainSubstring("start_time must be before")))
		})

		It("Should update the fields the mask names", func(ctx SpecContext) {
			created := MustSucceed(createRun(ctx, &runsv2.CreateAdhocRunRequest{
				Tags: []string{"cold"},
			})).Run
			MustSucceed(runs.UpdateRun(ctx, &runsv2.UpdateRunRequest{
				Run: &runsv2.Run{
					RunId: created.RunId,
					Name:  "renamed",
					Tags:  []string{"hot"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
			}))
			Expect(server.Runs()).To(HaveExactElements(SatisfyAll(
				HaveField("Name", "renamed"),
				HaveField("Tags", ConsistOf("cold")),
			)))
		})

		DescribeTable("Should reject an invalid update",
			func(ctx SpecContext, run *runsv2.Run, paths []string, code codes.Code) {
				created := MustSucceed(createRun(ctx, &runsv2.CreateAdhocRunRequest{}))
				if run.RunId == "" {
					run.RunId = created.Run.RunId
				}
				Expect(runs.UpdateRun(ctx, &runsv2.UpdateRunRequest{
					Run:        run,
					UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
				})).Error().To(WithTransform(status.Code, Equal(code)))
			},
			Entry("missing run", &runsv2.Run{RunId: "run-9"}, []string{"name"},
				codes.NotFound),
			Entry("client key", &runsv2.Run{}, []string{"client_key"},
				codes.InvalidArgument),
			Entry("stop before start", &runsv2.Run{StopTime: start},
				[]string{"stop_time"}, codes.InvalidArgument),
		)

		It("Should return the errors FailRuns sets, in order", func(ctx SpecContext) {
			server.FailRuns(status.Error(codes.Unavailable, "down"))
			Expect(createRun(ctx, &runsv2.CreateAdhocRunRequest{})).Error().
				To(WithTransform(status.Code, Equal(codes.Unavailable)))
			MustSucceed(createRun(ctx, &runsv2.CreateAdhocRunRequest{}))
		})
	})

	Describe("APIKey", func() {
		BeforeEach(func() { open(fake.Config{APIKey: "secret"}) })

		It("Should reject a call without the API key", func(ctx SpecContext) {
			Expect(configs.ListIngestionConfigs(
				ctx,
				&ingestionconfigsv2.ListIngestionConfigsRequest{},
			)).Error().To(MatchError(ContainSubstring("invalid API key")))
			Expect(sendAll(ctx)).To(MatchError(ContainSubstring("invalid API key")))
		})

		It("Should accept a call with the API key", func(ctx SpecContext) {
			authCtx := metadata.AppendToOutgoingContext(
				ctx,
				"authorization",
				"Bearer secret",
			)
			MustSucceed(configs.ListIngestionConfigs(
				authCtx,
				&ingestionconfigsv2.ListIngestionConfigsRequest{},
			))
		})
	})
})
