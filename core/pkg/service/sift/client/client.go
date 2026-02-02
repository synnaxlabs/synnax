// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package client provides the Sift client interface and implementations.
package client

import (
	"context"
	"io"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	metadatav1 "github.com/sift-stack/sift/go/gen/sift/metadata/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/confluence"
)

type (
	ChannelConfig                     = ingestionconfigsv1.ChannelConfig
	FlowConfig                        = ingestionconfigsv1.FlowConfig
	IngestionConfig                   = ingestionconfigsv1.IngestionConfig
	CreateIngestionConfigRequest      = ingestionconfigsv1.CreateIngestionConfigRequest
	CreateIngestionConfigResponse     = ingestionconfigsv1.CreateIngestionConfigResponse
	Run                               = runsv2.Run
	CreateRunRequest                  = runsv2.CreateRunRequest
	CreateRunResponse                 = runsv2.CreateRunResponse
	IngestWithConfigDataStreamRequest = ingestv1.IngestWithConfigDataStreamRequest
	MetadataValue                     = metadatav1.MetadataValue
)

// Ingester is a confluence sink for streaming data to Sift.
type Ingester interface {
	confluence.Sink[*IngestWithConfigDataStreamRequest]
	io.Closer
}

// Client is the interface for Sift operations.
type Client interface {
	// CreateIngestionConfig creates a new ingestion configuration.
	CreateIngestionConfig(
		context.Context,
		*CreateIngestionConfigRequest,
	) (*CreateIngestionConfigResponse, error)
	// CreateRun creates a new run.
	CreateRun(context.Context, *CreateRunRequest) (*CreateRunResponse, error)
	// OpenIngester opens an ingester for streaming data.
	OpenIngester(context.Context) (Ingester, error)
	// Close closes the client connection.
	Close() error
}

// Factory creates Client instances.
type Factory func(_ context.Context, uri, apiKey string) (Client, error)
