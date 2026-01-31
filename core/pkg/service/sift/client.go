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

// Client wraps the Sift gRPC client.
type Client struct {
	conn         grpc.SiftChannel
	props        DeviceProperties
	ingestionSvc ingestion_configsv1.IngestionConfigServiceClient
	ingestSvc    ingestv1.IngestServiceClient
	runSvc       runsv2.RunServiceClient
}

// NewClient creates a new Sift client.
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

// ChannelConfig represents a channel in a Sift flow.
type ChannelConfig struct {
	Name     string
	DataType typev1.ChannelDataType
}

// FlowConfig represents a Sift flow with its channels.
type FlowConfig struct {
	Name     string
	Channels []ChannelConfig
}

// GetOrCreateIngestionConfig retrieves or creates an ingestion config.
func (c *Client) GetOrCreateIngestionConfig(
	ctx context.Context,
	flows []FlowConfig,
) (*ingestion_configsv1.IngestionConfig, error) {
	clientKey := c.props.clientKey()

	// Try to find existing config
	listRes, err := c.ingestionSvc.ListIngestionConfigs(
		ctx,
		&ingestion_configsv1.ListIngestionConfigsRequest{
			Filter: fmt.Sprintf("client_key == '%s'", clientKey),
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list ingestion configs")
	}
	if listRes != nil && len(listRes.IngestionConfigs) > 0 {
		return listRes.IngestionConfigs[0], nil
	}

	// Create new config
	protoFlows := make([]*ingestion_configsv1.FlowConfig, len(flows))
	for i, flow := range flows {
		channels := make([]*ingestion_configsv1.ChannelConfig, len(flow.Channels))
		for j, ch := range flow.Channels {
			channels[j] = &ingestion_configsv1.ChannelConfig{
				Name:     ch.Name,
				DataType: ch.DataType,
			}
		}
		protoFlows[i] = &ingestion_configsv1.FlowConfig{
			Name:     flow.Name,
			Channels: channels,
		}
	}

	createRes, err := c.ingestionSvc.CreateIngestionConfig(
		ctx,
		&ingestion_configsv1.CreateIngestionConfigRequest{
			AssetName: c.props.AssetName,
			ClientKey: clientKey,
			Flows:     protoFlows,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ingestion config")
	}
	return createRes.IngestionConfig, nil
}

// CreateRun creates a new Sift run.
func (c *Client) CreateRun(ctx context.Context, name string) (*runsv2.Run, error) {
	createRes, err := c.runSvc.CreateRun(ctx, &runsv2.CreateRunRequest{
		Name:      name,
		StartTime: timestamppb.Now(),
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
	return &IngestStream{stream: stream, configID: ingestionConfigID}, nil
}

// Send sends data to the ingestion stream.
func (s *IngestStream) Send(req *ingestv1.IngestWithConfigDataStreamRequest) error {
	req.IngestionConfigId = s.configID
	return s.stream.Send(req)
}

// Close closes the ingestion stream.
func (s *IngestStream) Close() error {
	_, err := s.stream.CloseAndRecv()
	return err
}

// ClientPool manages shared Sift connections.
type ClientPool struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewClientPool creates a new client pool.
func NewClientPool() *ClientPool {
	return &ClientPool{clients: make(map[string]*Client)}
}

// Get retrieves or creates a client for the given device properties.
func (p *ClientPool) Get(ctx context.Context, props DeviceProperties) (*Client, error) {
	key := props.URI + ":" + props.AssetName

	p.mu.RLock()
	if client, ok := p.clients[key]; ok {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after write lock
	if client, ok := p.clients[key]; ok {
		return client, nil
	}

	client, err := NewClient(ctx, props)
	if err != nil {
		return nil, err
	}
	p.clients[key] = client
	return client, nil
}

// Close closes all connections.
func (p *ClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	errors.NewCatcher(errors.WithAggregation())

	var errs error
	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			errs = errors.Combine(errs, err)
		}
	}
	clear(p.clients)
	return errs
}
