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
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/confluence"
)

// Ingester is a confluence sink for streaming data to Sift.
type Ingester interface {
	confluence.Sink[*ingestv1.IngestWithConfigDataStreamRequest]
	io.Closer
}

// Client is the interface for Sift operations.
type Client interface {
	// CreateIngestionConfig creates a new ingestion configuration.
	CreateIngestionConfig(
		context.Context,
		*ingestionconfigsv1.CreateIngestionConfigRequest,
	) (*ingestionconfigsv1.CreateIngestionConfigResponse, error)
	// CreateRun creates a new run.
	CreateRun(
		context.Context,
		*runsv2.CreateRunRequest,
	) (*runsv2.CreateRunResponse, error)
	// OpenIngester opens an ingester for streaming data.
	OpenIngester(context.Context) (Ingester, error)
	// Close closes the client connection.
	Close() error
}

// Factory creates Client instances.
type Factory func(_ context.Context, uri, apiKey string) (Client, error)
