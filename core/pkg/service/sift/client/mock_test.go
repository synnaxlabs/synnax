// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package client_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	metadatav1 "github.com/sift-stack/sift/go/gen/sift/metadata/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Mock", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("MockFactoryConfig", func() {
		Describe("Validate", func() {
			It("Should return nil for empty config", func() {
				cfg := client.MockFactoryConfig{}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("Override", func() {
			It("Should override ErrorOnNew", func() {
				base := client.MockFactoryConfig{}
				override := client.MockFactoryConfig{ErrorOnNew: config.True()}
				result := base.Override(override)
				Expect(*result.ErrorOnNew).To(BeTrue())
			})

			It("Should override ErrorOnCreateIngestionConfig", func() {
				base := client.MockFactoryConfig{}
				override := client.MockFactoryConfig{
					ErrorOnCreateIngestionConfig: config.True(),
				}
				result := base.Override(override)
				Expect(*result.ErrorOnCreateIngestionConfig).To(BeTrue())
			})

			It("Should override ErrorOnCreateRun", func() {
				base := client.MockFactoryConfig{}
				override := client.MockFactoryConfig{ErrorOnCreateRun: config.True()}
				result := base.Override(override)
				Expect(*result.ErrorOnCreateRun).To(BeTrue())
			})

			It("Should not override when other is nil", func() {
				base := client.MockFactoryConfig{ErrorOnNew: config.True()}
				override := client.MockFactoryConfig{}
				result := base.Override(override)
				Expect(*result.ErrorOnNew).To(BeTrue())
			})
		})
	})

	Describe("MockFactory", func() {
		Describe("New", func() {
			It("Should create a client factory", func() {
				factory := MustSucceed(client.NewMockFactory())
				Expect(factory(ctx, "", "")).ToNot(BeNil())
			})
			It("Should return error when ErrorOnNew is set", func() {
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					ErrorOnNew: config.True(),
				}))
				Expect(factory(ctx, "", "")).Error().
					To(MatchError(ContainSubstring("failed to create client")))
			})
		})
	})

	Describe("Client", func() {
		Describe("CreateIngestionConfig", func() {
			It("Should return a config ID from CreateIngestionConfig", func() {
				factory := MustSucceed(client.NewMockFactory())
				c := MustSucceed(factory(ctx, "", ""))
				res := MustSucceed(c.CreateIngestionConfig(
					ctx,
					&ingestionconfigsv1.CreateIngestionConfigRequest{
						ClientKey: "test-key",
						AssetName: "test-asset",
					},
				)).IngestionConfig
				Expect(res.GetClientKey()).To(Equal("test-key"))
				Expect(res.GetAssetId()).To(Equal("test-asset"))
				Expect(res.GetIngestionConfigId()).ToNot(BeEmpty())
			})

			It("Should return error from CreateIngestionConfig when configured", func() {
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					ErrorOnCreateIngestionConfig: config.True(),
				}))
				c := MustSucceed(factory(ctx, "", ""))
				Expect(c.CreateIngestionConfig(
					ctx,
					&ingestionconfigsv1.CreateIngestionConfigRequest{},
				)).Error().To(MatchError(
					ContainSubstring("failed to create ingestion config"),
				))
			})
		})
		Describe("CreateRun", func() {
			It("Should return a run ID from CreateRun", func() {
				factory := MustSucceed(client.NewMockFactory())
				c := MustSucceed(factory(ctx, "", ""))
				startTime := timestamppb.Now()
				stopTime := timestamppb.New(startTime.AsTime().Add(time.Minute))
				clientKey := "test-key"
				tags := []string{"test-tag"}
				metadata := []*metadatav1.MetadataValue{
					{
						Key: &metadatav1.MetadataKey{
							Name: "test-key",
							Type: metadatav1.MetadataKeyType_METADATA_KEY_TYPE_STRING,
						},
						Value: &metadatav1.MetadataValue_StringValue{
							StringValue: "test-value",
						},
					},
				}
				res := MustSucceed(c.CreateRun(
					ctx,
					&runsv2.CreateRunRequest{
						ClientKey:      &clientKey,
						Name:           "test-run",
						StartTime:      startTime,
						StopTime:       stopTime,
						OrganizationId: "test-organization",
						Tags:           tags,
						Description:    "test-description",
						Metadata:       metadata,
					},
				)).Run
				Expect(res.GetRunId()).ToNot(BeEmpty())
				Expect(res.GetClientKey()).To(Equal(clientKey))
				Expect(res.GetStartTime()).To(Equal(startTime))
				Expect(res.GetStopTime()).To(Equal(stopTime))
				Expect(res.GetOrganizationId()).To(Equal("test-organization"))
				Expect(res.GetTags()).To(Equal(tags))
				Expect(res.GetDescription()).To(Equal("test-description"))
				Expect(res.GetMetadata()).To(Equal(metadata))
			})

			It("Should return error from CreateRun when configured", func() {
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					ErrorOnCreateRun: config.True(),
				}))
				c := MustSucceed(factory(ctx, "", ""))
				Expect(c.CreateRun(ctx, &runsv2.CreateRunRequest{})).Error().
					To(MatchError(ContainSubstring("failed to create run")))
			})
		})
		Describe("OpenIngester", func() {
			It("Should return ingester by default", func() {
				factory := MustSucceed(client.NewMockFactory())
				c := MustSucceed(factory(ctx, "", ""))
				ingester := MustSucceed(c.OpenIngester(ctx))
				Expect(ingester).ToNot(BeNil())
				Expect(ingester.Close()).To(Succeed())
			})
			It("Should return error from OpenIngester when configured", func() {
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					ErrorOnOpenIngester: config.True(),
				}))
				c := MustSucceed(factory(ctx, "", ""))
				Expect(c.OpenIngester(ctx)).Error().
					To(MatchError(ContainSubstring("failed to open ingester")))
			})

			It("Should pipe requests to configured inlet", func() {
				requests := confluence.
					NewStream[*ingestv1.IngestWithConfigDataStreamRequest](1)
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					Requests: requests,
				}))
				c := MustSucceed(factory(ctx, "", ""))
				ingester := MustSucceed(c.OpenIngester(ctx))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				input := confluence.
					NewStream[*ingestv1.IngestWithConfigDataStreamRequest](1)
				ingester.InFrom(input)
				ingester.Flow(sCtx)

				timestamp := telem.Now().Time()

				request := ingestv1.IngestWithConfigDataStreamRequest{
					Flow:      "flow1",
					Timestamp: timestamppb.New(timestamp),
					RunId:     "run-id",
				}
				input.Inlet() <- &request
				Expect(<-requests.Outlet()).To(Equal(&request))
			})
			It("Should do nothing when Requests is not configured", func() {
				factory := MustSucceed(client.NewMockFactory())
				c := MustSucceed(factory(ctx, "", ""))
				ingester := MustSucceed(c.OpenIngester(ctx))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				input := confluence.
					NewStream[*ingestv1.IngestWithConfigDataStreamRequest](1)
				ingester.InFrom(input)
				ingester.Flow(sCtx)
				request := &ingestv1.IngestWithConfigDataStreamRequest{
					Flow:      "flow1",
					Timestamp: timestamppb.Now(),
					RunId:     "run-id",
				}
				input.Inlet() <- request
				Expect(ingester.Close()).To(Succeed())
			})

			It("Should return error from ingester Close when configured", func() {
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					ErrorOnIngesterClose: config.True(),
				}))
				c := MustSucceed(factory(ctx, "", ""))
				ingester := MustSucceed(c.OpenIngester(ctx))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				input := confluence.
					NewStream[*ingestv1.IngestWithConfigDataStreamRequest](1)
				ingester.InFrom(input)
				ingester.Flow(sCtx)
				input.Inlet() <- &ingestv1.IngestWithConfigDataStreamRequest{
					Flow:      "flow1",
					Timestamp: timestamppb.Now(),
					RunId:     "run-id",
				}
				input.Close()
				Expect(ingester.Close()).Error().
					To(MatchError(ContainSubstring("failed to close ingester")))
			})
		})

		Describe("Close", func() {
			It("Should call OnClose when configured", func() {
				var closed bool
				factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
					OnClose: func() error { closed = true; return nil },
				}))
				c := MustSucceed(factory(ctx, "", ""))
				Expect(c.Close()).To(Succeed())
				Expect(closed).To(BeTrue())
			})
			It("Should do nothing when OnClose is not configured", func() {
				factory := MustSucceed(client.NewMockFactory())
				c := MustSucceed(factory(ctx, "", ""))
				Expect(c.Close()).To(Succeed())
			})
		})
	})
})
