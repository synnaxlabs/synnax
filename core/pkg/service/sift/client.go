// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"context"
	"fmt"
	"sync"

	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestion_configsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/sift-stack/sift/go/grpc"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client wraps the Sift gRPC client for connection management.
type Client struct {
	conn         grpc.SiftChannel
	props        DeviceProperties
	ingestionSvc ingestion_configsv1.IngestionConfigServiceClient
	ingestSvc    ingestv1.IngestServiceClient
	runSvc       runsv2.RunServiceClient
}

// NewClient creates a new Sift client from device properties.
func NewClient(ctx context.Context, props DeviceProperties) (*Client, error) {
	conn, err := grpc.UseSiftChannel(ctx, grpc.SiftChannelConfig{
		Uri:            props.URI,
		Apikey:         props.APIKey,
		UseSystemCerts: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Sift")
	}
	return &Client{
		conn:         conn,
		props:        props,
		ingestionSvc: ingestion_configsv1.NewIngestionConfigServiceClient(conn),
		ingestSvc:    ingestv1.NewIngestServiceClient(conn),
		runSvc:       runsv2.NewRunServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ChannelConfig represents a channel configuration for an ingestion config.
type ChannelConfig struct {
	Name        string
	Component   string
	Unit        string
	Description string
	DataType    typev1.ChannelDataType
}

// FlowConfig represents a flow configuration for an ingestion config.
type FlowConfig struct {
	Name     string
	Channels []ChannelConfig
}

// GetOrCreateIngestionConfig retrieves or creates an ingestion config.
func (c *Client) GetOrCreateIngestionConfig(
	ctx context.Context,
	flows []FlowConfig,
) (*ingestion_configsv1.IngestionConfig, error) {
	// First try to find existing config by client key
	listRes, err := c.ingestionSvc.ListIngestionConfigs(
		ctx,
		&ingestion_configsv1.ListIngestionConfigsRequest{
			Filter: fmt.Sprintf("client_key == '%s'", c.props.ClientKey),
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list ingestion configs")
	}
	if listRes != nil && len(listRes.IngestionConfigs) > 0 {
		return listRes.IngestionConfigs[0], nil
	}

	// Convert flow configs to Sift proto format
	protoFlows := make([]*ingestion_configsv1.FlowConfig, len(flows))
	for i, flow := range flows {
		channels := make([]*ingestion_configsv1.ChannelConfig, len(flow.Channels))
		for j, ch := range flow.Channels {
			channels[j] = &ingestion_configsv1.ChannelConfig{
				Name:        ch.Name,
				Component:   ch.Component,
				Unit:        ch.Unit,
				Description: ch.Description,
				DataType:    ch.DataType,
			}
		}
		protoFlows[i] = &ingestion_configsv1.FlowConfig{
			Name:     flow.Name,
			Channels: channels,
		}
	}

	// Create new ingestion config
	createRes, err := c.ingestionSvc.CreateIngestionConfig(
		ctx,
		&ingestion_configsv1.CreateIngestionConfigRequest{
			AssetName: c.props.AssetName,
			ClientKey: c.props.ClientKey,
			Flows:     protoFlows,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ingestion config")
	}
	return createRes.IngestionConfig, nil
}

// CreateRun creates a new run for grouping ingested data.
func (c *Client) CreateRun(ctx context.Context, name string) (*runsv2.Run, error) {
	ts := timestamppb.Now()
	createRes, err := c.runSvc.CreateRun(ctx, &runsv2.CreateRunRequest{
		Name:      name,
		StartTime: ts,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create run")
	}
	return createRes.Run, nil
}

// IngestStream represents an open ingestion stream.
type IngestStream struct {
	stream   ingestv1.IngestService_IngestWithConfigDataStreamClient
	configID string
	orgID    string
}

// OpenIngestStream opens a new ingestion stream.
func (c *Client) OpenIngestStream(
	ctx context.Context,
	ingestionConfigID string,
) (*IngestStream, error) {
	stream, err := c.ingestSvc.IngestWithConfigDataStream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open ingest stream")
	}
	return &IngestStream{
		stream:   stream,
		configID: ingestionConfigID,
		orgID:    c.props.OrganizationID,
	}, nil
}

// Send sends data to the ingestion stream.
func (s *IngestStream) Send(req *ingestv1.IngestWithConfigDataStreamRequest) error {
	req.IngestionConfigId = s.configID
	if s.orgID != "" {
		req.OrganizationId = s.orgID
	}
	return s.stream.Send(req)
}

// Close closes the ingestion stream and returns any error.
func (s *IngestStream) Close() error {
	_, err := s.stream.CloseAndRecv()
	return err
}

// ConnectionPool manages shared Sift connections keyed by device.
type ConnectionPool struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{clients: make(map[string]*Client)}
}

// Get retrieves or creates a client for the given device properties.
func (p *ConnectionPool) Get(ctx context.Context, props DeviceProperties) (*Client, error) {
	p.mu.RLock()
	if client, ok := p.clients[props.ClientKey]; ok {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := p.clients[props.ClientKey]; ok {
		return client, nil
	}

	client, err := NewClient(ctx, props)
	if err != nil {
		return nil, err
	}
	p.clients[props.ClientKey] = client
	return client, nil
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var combinedErr error
	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			combinedErr = errors.Combine(combinedErr, err)
		}
	}
	p.clients = make(map[string]*Client)
	return combinedErr
}
