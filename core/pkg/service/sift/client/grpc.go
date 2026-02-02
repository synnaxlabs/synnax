// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package client

import (
	"context"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	siftgrpc "github.com/sift-stack/sift/go/grpc"
	"github.com/synnaxlabs/x/confluence"
)

type grpc struct{ siftgrpc.SiftChannel }

var _ Client = (*grpc)(nil)

// NewGRPC creates a new Sift gRPC client.
func NewGRPC(ctx context.Context, uri, apiKey string) (Client, error) {
	conn, err := siftgrpc.UseSiftChannel(ctx, siftgrpc.SiftChannelConfig{
		Uri:            uri,
		Apikey:         apiKey,
		UseSystemCerts: true,
	})
	if err != nil {
		return nil, err
	}
	return &grpc{conn}, nil
}

var _ Factory = NewGRPC

func (g *grpc) CreateIngestionConfig(
	ctx context.Context,
	req *CreateIngestionConfigRequest,
) (*CreateIngestionConfigResponse, error) {
	return ingestionconfigsv1.
		NewIngestionConfigServiceClient(g).
		CreateIngestionConfig(ctx, req)
}

func (g *grpc) CreateRun(
	ctx context.Context,
	req *CreateRunRequest,
) (*CreateRunResponse, error) {
	return runsv2.NewRunServiceClient(g).CreateRun(ctx, req)
}

func (g *grpc) OpenIngester(ctx context.Context) (Ingester, error) {
	stream, err := ingestv1.NewIngestServiceClient(g).IngestWithConfigDataStream(ctx)
	if err != nil {
		return nil, err
	}
	return newGRPCIngester(stream), nil
}

type grpcIngester struct {
	confluence.UnarySink[*IngestWithConfigDataStreamRequest]
	stream ingestv1.IngestService_IngestWithConfigDataStreamClient
}

var _ Ingester = (*grpcIngester)(nil)

func newGRPCIngester(
	stream ingestv1.IngestService_IngestWithConfigDataStreamClient,
) *grpcIngester {
	i := &grpcIngester{stream: stream}
	i.UnarySink.Sink = i.sink
	return i
}

func (i *grpcIngester) sink(_ context.Context, req *IngestWithConfigDataStreamRequest) error {
	return i.stream.Send(req)
}

func (i *grpcIngester) Close() error { _, err := i.stream.CloseAndRecv(); return err }
