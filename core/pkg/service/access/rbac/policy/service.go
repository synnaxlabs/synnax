// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a policy service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store policies.
	DB *gorp.DB
	// Ontology is the ontology that the service will use to create relationships
	// between policies and other resources, such as roles, within the Synnax cluster.
	Ontology *ontology.Ontology
	// Signals is the signals provider that the service will use to publish changes to
	// policies.
	Signals *signals.Provider
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override overrides fields in c with the valid fields in other.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate validates the configuration.
func (c ServiceConfig) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

// Service is the main entrypoint for managing policies within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting policies. It also
// provides mechanisms for listening to changes in policies.
type Service struct {
	cfg            ServiceConfig
	signals        io.Closer
	systemPolicies []Policy
	mu             sync.RWMutex
}

var _ io.Closer = (*Service)(nil)

// OpenService opens a new policy service using the provided configuration. If error is
// nil, the service is ready for use and must be closed by calling Close in order to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.Signals != nil {
		if s.signals, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Policy](cfg.DB),
		); err != nil {
			return nil, err
		}
	}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// Close closes the policy service and releases any resources that it may have acquired.
func (s *Service) Close() error {
	if s.signals == nil {
		return nil
	}
	return s.signals.Close()
}

// AddSystemPolicies adds system policies to the service. System policies are applied to
// all subjects and are typically used for system-wide restrictions (e.g., preventing
// modification of system-managed resources). If a policy has a nil key, a new UUID will
// be generated for it.
func (s *Service) AddSystemPolicies(policies ...Policy) {
	for i := range policies {
		if policies[i].Key == uuid.Nil {
			policies[i].Key = uuid.New()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPolicies = append(s.systemPolicies, policies...)
}

// SystemPolicies returns all system policies that have been added to the service.
func (s *Service) SystemPolicies() []Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemPolicies
}
