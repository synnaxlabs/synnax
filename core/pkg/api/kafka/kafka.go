// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package kafka serves the Kafka routes: connection tests, and the browse and sample of
// the topics of a cluster device.
package kafka

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the Kafka API service.
type ServiceConfig struct {
	// Device retrieves cluster devices.
	// [REQUIRED]
	Device *device.Service
	// RBAC enforces access control on every request.
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Device = override.Nil(c.Device, other.Device)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.kafka")
	v.NotNil("device", c.Device)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service serves the Kafka routes.
type Service struct{ cfg ServiceConfig }

// NewService returns a Service. It returns a validation error when the configuration is
// invalid.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

func (s *Service) enforce(
	ctx context.Context,
	action access.Action,
	id ontology.ID,
) error {
	return s.cfg.RBAC.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: []ontology.ID{id},
	})
}

// properties returns the connection properties of the cluster device key. It requires
// retrieve access on the device.
func (s *Service) properties(
	ctx context.Context,
	key device.Key,
) (kafka.Properties, error) {
	if err := s.enforce(
		ctx,
		access.ActionRetrieve,
		device.OntologyID(key),
	); err != nil {
		return kafka.Properties{}, err
	}
	var dev device.Device
	if err := s.cfg.Device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return kafka.Properties{}, err
	}
	return kafka.DecodeProperties(dev)
}

type (
	// TestRequest carries the cluster to test a connection to.
	TestRequest struct {
		// Device is the key of the cluster device that the test edits. Empty for a
		// cluster that is not a device yet.
		Device device.Key `json:"device" msgpack:"device"`
		// Properties are the connection properties to test.
		Properties kafka.Properties `json:"properties" msgpack:"properties"`
	}
	// TestResponse carries the outcome of each stage of the test.
	TestResponse struct {
		// Steps are the stages that ran, in order. The last is the first that failed.
		Steps []driver.TestStep `json:"steps" msgpack:"steps"`
	}
)

// Test tests a connection to the cluster of req. It requires update access on the
// device that it edits, or create access on devices for a cluster that is not a device
// yet.
func (s *Service) Test(ctx context.Context, req TestRequest) (TestResponse, error) {
	id, action := ontology.ID{Type: ontology.ResourceTypeDevice}, access.ActionCreate
	if req.Device != "" {
		id, action = device.OntologyID(req.Device), access.ActionUpdate
	}
	if err := s.enforce(ctx, action, id); err != nil {
		return TestResponse{}, err
	}
	steps, err := kafka.TestConnection(ctx, req.Properties)
	return TestResponse{Steps: steps}, err
}

type (
	// BrowseRequest selects the cluster to browse.
	BrowseRequest struct {
		// Device is the key of the cluster device.
		Device device.Key `json:"device" msgpack:"device"`
	}
	// BrowseResponse carries the topics of the cluster.
	BrowseResponse struct {
		// Topics are the topics of the cluster, sorted by name.
		Topics []kafka.Topic `json:"topics" msgpack:"topics"`
	}
)

// Browse lists the topics of a cluster device. It requires retrieve access on the
// device.
func (s *Service) Browse(
	ctx context.Context,
	req BrowseRequest,
) (BrowseResponse, error) {
	props, err := s.properties(ctx, req.Device)
	if err != nil {
		return BrowseResponse{}, err
	}
	topics, err := kafka.Browse(ctx, props)
	return BrowseResponse{Topics: topics}, err
}

type (
	// SampleRequest selects the records to sample.
	SampleRequest struct {
		// Device is the key of the cluster device.
		Device device.Key `json:"device" msgpack:"device"`
		// Topics are the topics to read.
		Topics []string `json:"topics" msgpack:"topics"`
		// Count is the most records to read from each topic. Zero selects 100, and the
		// most is 1,000.
		Count int `json:"count" msgpack:"count"`
	}
	// SampleResponse carries the sample of each topic.
	SampleResponse struct {
		// Topics are the samples, in the order of the topics of the request.
		Topics []kafka.SampledTopic `json:"topics" msgpack:"topics"`
	}
)

// Sample reads records from topics of a cluster device and infers their shapes. It
// requires retrieve access on the device.
func (s *Service) Sample(
	ctx context.Context,
	req SampleRequest,
) (SampleResponse, error) {
	props, err := s.properties(ctx, req.Device)
	if err != nil {
		return SampleResponse{}, err
	}
	topics, err := kafka.Sample(ctx, props, kafka.SampleRequest{
		Topics: req.Topics,
		Count:  req.Count,
	})
	return SampleResponse{Topics: topics}, err
}
