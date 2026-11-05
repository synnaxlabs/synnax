// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package mqtt serves the MQTT routes: connection tests, and the browse and sample of
// the topics of a broker device.
package mqtt

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the MQTT API service.
type ServiceConfig struct {
	// MQTT reads brokers. It shares the connections of the MQTT tasks.
	// [REQUIRED]
	MQTT *mqtt.Factory
	// RBAC enforces access control on every request.
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.MQTT = override.Nil(c.MQTT, other.MQTT)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.mqtt")
	v.NotNil("mqtt", c.MQTT)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service serves the MQTT routes.
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

type (
	TestRequest = mqtt.TestRequest
	// TestResponse carries the outcome of each stage of a connection test.
	TestResponse struct {
		// Steps are the stages that ran, in order. The last is the first that failed.
		Steps []driver.TestStep `json:"steps" msgpack:"steps"`
	}
	BrowseRequest  = mqtt.BrowseRequest
	BrowseResponse = mqtt.BrowseResult
	SampleRequest  = mqtt.SampleRequest
	SampleResponse = mqtt.SampleResult
)

// Test tests a connection to the broker of req. It requires update access on the device
// that it edits, or create access on devices for a broker that is not a device yet.
func (s *Service) Test(ctx context.Context, req TestRequest) (TestResponse, error) {
	id, action := ontology.ID{Type: ontology.ResourceTypeDevice}, access.ActionCreate
	if req.Device != "" {
		id, action = device.OntologyID(req.Device), access.ActionUpdate
	}
	if err := s.enforce(ctx, action, id); err != nil {
		return TestResponse{}, err
	}
	steps, err := s.cfg.MQTT.TestConnection(ctx, req)
	return TestResponse{Steps: steps}, err
}

// Browse lists the topics that a broker device delivers while the Core listens. It
// requires retrieve access on the device.
func (s *Service) Browse(
	ctx context.Context,
	req BrowseRequest,
) (BrowseResponse, error) {
	if err := s.enforce(
		ctx,
		access.ActionRetrieve,
		device.OntologyID(req.Device),
	); err != nil {
		return BrowseResponse{}, err
	}
	return s.cfg.MQTT.Browse(ctx, req)
}

// Sample reads messages from topics of a broker device and infers their shapes. It
// requires retrieve access on the device.
func (s *Service) Sample(
	ctx context.Context,
	req SampleRequest,
) (SampleResponse, error) {
	if err := s.enforce(
		ctx,
		access.ActionRetrieve,
		device.OntologyID(req.Device),
	); err != nil {
		return SampleResponse{}, err
	}
	return s.cfg.MQTT.Sample(ctx, req)
}
