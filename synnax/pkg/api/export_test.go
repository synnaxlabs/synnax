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
	"encoding/csv"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Export", Ordered, func() {
	var (
		cluster      *mock.Cluster
		node         mock.Node
		serviceLayer *service.Layer
		apiProvider  api.Provider
		indexCh      channel.Channel
		dataCh       channel.Channel
	)
	BeforeAll(func() {
		cluster = mock.ProvisionCluster(ctx, 1)
		node = cluster.Nodes[1]
		securityProvider := MustSucceed(security.NewProvider(security.ProviderConfig{}))
		serviceLayer = MustSucceed(service.Open(ctx, service.Config{
			Distribution: node.Layer,
			Security:     securityProvider,
		}))
		apiProvider = api.NewProvider(api.Config{
			Distribution: node.Layer,
			Service:      serviceLayer,
		})
		indexCh = channel.Channel{
			Name:     "time",
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}
		Expect(node.Channel.Create(ctx, &indexCh)).To(Succeed())
		dataCh = channel.Channel{
			Name:       "data",
			DataType:   telem.Float64T,
			LocalIndex: indexCh.LocalKey,
		}
		Expect(node.Channel.Create(ctx, &dataCh)).To(Succeed())
		w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
			Keys: []channel.Key{dataCh.Key(), indexCh.Key()},
		}))
		Expect(w.Write(
			frame.NewMulti(
				[]channel.Key{dataCh.Key(), indexCh.Key()},
				[]telem.Series{
					telem.NewSeriesV[float64](10, 20, 30, 40, 50),
					telem.NewSeriesV[telem.TimeStamp](1, 2, 3, 4, 5),
				},
			),
		)).Error().To(Succeed())
		Expect(w.Commit()).Error().To(Succeed())
		Expect(w.Close()).To(Succeed())
	})

	Describe("CSV", func() {
		It("should return an error in the reader if the request has no keys", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{}))
			_, err := io.ReadAll(reader)
			Expect(err).To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
		It("should export a single channel", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys: []channel.Key{dataCh.Key(), indexCh.Key()},
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{{"data", "time"}}))
		})
		It("should export with data and index channels", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys:      []channel.Key{dataCh.Key(), indexCh.Key()},
				TimeRange: telem.TimeRangeMax,
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{
				{"data", "time"},
				{"10", "1"},
				{"20", "2"},
				{"30", "3"},
				{"40", "4"},
				{"50", "5"},
			}))
		})
		It("should export the index channel if it is not specified", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys: []channel.Key{dataCh.Key()},
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{{"data", "time"}}))
		})
		It("should rename channels if the channel names are provided", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys: []channel.Key{dataCh.Key()},
				ChannelNames: map[channel.Key]string{
					dataCh.Key():  "renamed_data",
					indexCh.Key(): "renamed_time",
				},
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{{
				"renamed_data", "renamed_time",
			}}))
		})
		It("should allow partial renaming of channels", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys:         []channel.Key{dataCh.Key()},
				ChannelNames: map[channel.Key]string{dataCh.Key(): "renamed_data"},
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{{"renamed_data", "time"}}))
		})
		It("should allow reading partial time ranges", func() {
			svc := api.NewExportService(apiProvider)
			reader := MustSucceed(svc.ExportCSV(ctx, api.ExportCSVRequest{
				Keys:      []channel.Key{dataCh.Key()},
				TimeRange: telem.TimeRange{Start: 2, End: 4},
			}))
			csvReader := csv.NewReader(reader)
			Expect(csvReader.ReadAll()).To(Equal([][]string{
				{"data", "time"},
				{"20", "2"},
				{"30", "3"},
			}))
		})
	})

	AfterAll(func() {
		Expect(serviceLayer.Close()).To(Succeed())
		Expect(cluster.Close()).To(Succeed())
	})
})
