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

	"github.com/google/uuid"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
)

// MockFactoryConfig configures a mock Client for testing.
type MockFactoryConfig struct {
	// ErrorOnNew causes the factory to return an error when creating the client.
	ErrorOnNew *bool
	// ErrorOnCreateIngestionConfig causes CreateIngestionConfig to return an error.
	ErrorOnCreateIngestionConfig *bool
	// ErrorOnCreateRun causes CreateRun to return an error.
	ErrorOnCreateRun *bool
	// ErrorOnOpenIngester causes OpenIngester to return an error.
	ErrorOnOpenIngester *bool
	// ErrorOnIngesterClose means that the ingester will return an error when Close is
	// called.
	ErrorOnIngesterClose *bool
	// OnClose is called when Close is invoked.
	OnClose func() error
	// Requests receives all requests sent to the ingest stream.
	Requests confluence.Inlet[*ingestv1.IngestWithConfigDataStreamRequest]
}

var _ config.Config[MockFactoryConfig] = MockFactoryConfig{}

// Override implements config.Config.
func (c MockFactoryConfig) Override(other MockFactoryConfig) MockFactoryConfig {
	c.ErrorOnNew = override.Nil(c.ErrorOnNew, other.ErrorOnNew)
	c.ErrorOnCreateIngestionConfig = override.Nil(
		c.ErrorOnCreateIngestionConfig,
		other.ErrorOnCreateIngestionConfig,
	)
	c.ErrorOnCreateRun = override.Nil(c.ErrorOnCreateRun, other.ErrorOnCreateRun)
	c.ErrorOnOpenIngester = override.Nil(
		c.ErrorOnOpenIngester,
		other.ErrorOnOpenIngester,
	)
	c.ErrorOnIngesterClose = override.Nil(
		c.ErrorOnIngesterClose,
		other.ErrorOnIngesterClose,
	)
	c.OnClose = override.Nil(c.OnClose, other.OnClose)
	c.Requests = override.Nil(c.Requests, other.Requests)
	return c
}

// Validate implements config.Config.
func (MockFactoryConfig) Validate() error { return nil }

type mock struct{ MockFactoryConfig }

var _ Client = (*mock)(nil)

// NewMockFactory creates a Factory that returns mock Clients for testing.
func NewMockFactory(cfgs ...MockFactoryConfig) (Factory, error) {
	cfg, err := config.New(MockFactoryConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return func(context.Context, string, string) (Client, error) {
		if cfg.ErrorOnNew != nil && *cfg.ErrorOnNew {
			return nil, errors.New("failed to create client")
		}
		return &mock{MockFactoryConfig: cfg}, nil
	}, nil
}

func (m *mock) CreateIngestionConfig(
	_ context.Context,
	req *ingestionconfigsv1.CreateIngestionConfigRequest,
) (*ingestionconfigsv1.CreateIngestionConfigResponse, error) {
	if m.ErrorOnCreateIngestionConfig != nil &&
		*m.ErrorOnCreateIngestionConfig {
		return nil, errors.New("failed to create ingestion config")
	}
	return &ingestionconfigsv1.CreateIngestionConfigResponse{
		IngestionConfig: &ingestionconfigsv1.IngestionConfig{
			IngestionConfigId: uuid.New().String(),
			AssetId:           req.AssetName,
			ClientKey:         req.ClientKey,
		},
	}, nil
}

func (m *mock) CreateRun(
	_ context.Context,
	req *runsv2.CreateRunRequest,
) (*runsv2.CreateRunResponse, error) {
	if m.ErrorOnCreateRun != nil && *m.ErrorOnCreateRun {
		return nil, errors.New("failed to create run")
	}
	return &runsv2.CreateRunResponse{
		Run: &runsv2.Run{
			RunId:          uuid.New().String(),
			ClientKey:      req.ClientKey,
			StartTime:      req.StartTime,
			StopTime:       req.StopTime,
			OrganizationId: req.OrganizationId,
			Name:           req.Name,
			Description:    req.Description,
			Tags:           req.Tags,
			Metadata:       req.Metadata,
		},
	}, nil
}

func (m *mock) OpenIngester(context.Context) (Ingester, error) {
	if m.ErrorOnOpenIngester != nil && *m.ErrorOnOpenIngester {
		return nil, errors.New("failed to open ingester")
	}
	var errorOnStreamClose bool
	if m.ErrorOnIngesterClose != nil {
		errorOnStreamClose = *m.ErrorOnIngesterClose
	}
	return newMockIngester(m.Requests, errorOnStreamClose), nil
}

func (m *mock) Close() error {
	if m.OnClose != nil {
		return m.OnClose()
	}
	return nil
}

type mockIngester struct {
	confluence.UnarySink[*ingestv1.IngestWithConfigDataStreamRequest]
	requests           confluence.Inlet[*ingestv1.IngestWithConfigDataStreamRequest]
	errorOnStreamClose bool
}

var _ Ingester = (*mockIngester)(nil)

func newMockIngester(
	requests confluence.Inlet[*ingestv1.IngestWithConfigDataStreamRequest],
	errorOnStreamClose bool,
) *mockIngester {
	s := &mockIngester{requests: requests, errorOnStreamClose: errorOnStreamClose}
	s.UnarySink.Sink = s.sink
	return s
}

func (i *mockIngester) sink(
	ctx context.Context,
	req *ingestv1.IngestWithConfigDataStreamRequest,
) error {
	if i.requests == nil {
		return nil
	}
	return signal.SendUnderContext(
		ctx,
		i.requests.Inlet(),
		req,
	)
}

func (i *mockIngester) Close() error {
	if i.errorOnStreamClose {
		return errors.New("failed to close ingester")
	}
	return nil
}
