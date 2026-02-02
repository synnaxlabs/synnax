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
	Requests confluence.Inlet[*DataStreamRequest]
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
func (c MockFactoryConfig) Validate() error { return nil }

type mock struct{ cfg MockFactoryConfig }

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
		return &mock{cfg: cfg}, nil
	}, nil
}

func (m *mock) CreateIngestionConfig(
	_ context.Context,
	req *CreateIngestionConfigRequest,
) (*CreateIngestionConfigResponse, error) {
	if m.cfg.ErrorOnCreateIngestionConfig != nil &&
		*m.cfg.ErrorOnCreateIngestionConfig {
		return nil, errors.New("failed to create ingestion config")
	}
	return &CreateIngestionConfigResponse{IngestionConfig: &IngestionConfig{
		IngestionConfigId: uuid.New().String(),
		AssetId:           req.AssetName,
		ClientKey:         req.ClientKey,
	}}, nil
}

func (m *mock) CreateRun(
	_ context.Context,
	req *CreateRunRequest,
) (*CreateRunResponse, error) {
	if m.cfg.ErrorOnCreateRun != nil && *m.cfg.ErrorOnCreateRun {
		return nil, errors.New("failed to create run")
	}
	return &CreateRunResponse{
		Run: &Run{
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
	if m.cfg.ErrorOnOpenIngester != nil && *m.cfg.ErrorOnOpenIngester {
		return nil, errors.New("failed to open ingester")
	}
	var errorOnStreamClose bool
	if m.cfg.ErrorOnIngesterClose != nil {
		errorOnStreamClose = *m.cfg.ErrorOnIngesterClose
	}
	return newMockIngester(m.cfg.Requests, errorOnStreamClose), nil
}

func (m *mock) Close() error {
	if m.cfg.OnClose != nil {
		return m.cfg.OnClose()
	}
	return nil
}

type mockIngester struct {
	confluence.UnarySink[*DataStreamRequest]
	requests           confluence.Inlet[*DataStreamRequest]
	errorOnStreamClose bool
}

var _ Ingester = (*mockIngester)(nil)

func newMockIngester(
	requests confluence.Inlet[*DataStreamRequest],
	errorOnStreamClose bool,
) *mockIngester {
	s := &mockIngester{requests: requests, errorOnStreamClose: errorOnStreamClose}
	s.UnarySink.Sink = s.sink
	return s
}

func (s *mockIngester) sink(ctx context.Context, req *DataStreamRequest) error {
	if s.requests == nil {
		return nil
	}
	return signal.SendUnderContext(
		ctx,
		s.requests.Inlet(),
		req,
	)
}

func (s *mockIngester) Close() error {
	if s.errorOnStreamClose {
		return errors.New("failed to close ingester")
	}
	return nil
}
