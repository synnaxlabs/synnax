// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening an import/export service.
type ServiceConfig struct {
	// DB is the database used to wrap import operations in a single transaction.
	//
	// [REQUIRED]
	DB *gorp.DB
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("imex")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Service is the central import/export registry. Handlers are registered via
// RegisterImporterExporter / RegisterImporter / RegisterExporter  and routed by their
// Type.
type Service struct {
	cfg       ServiceConfig
	importers map[string]Importer
	exporters map[ontology.ResourceType]Exporter
}

// NewService creates a new, empty import/export registry. Handlers register themselves
// via RegisterImporterExporter / RegisterImporter / RegisterExporter at their own
// registration time, typically by accepting the Service in their own service config.
func NewService(_ context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:       cfg,
		importers: make(map[string]Importer),
		exporters: make(map[ontology.ResourceType]Exporter),
	}, nil
}

// RegisterImporterExporter registers a single handler for both halves under
// its own Type. Use this for symmetric services (one resource type for both
// import and export) — e.g., logs, schematics.
func (s *Service) RegisterImporterExporter(ie ImportExporter) {
	t := ie.Type()
	s.importers[string(t)] = ie
	s.exporters[t] = ie
}

// RegisterImporter adds an Importer for the given resource type. Use this for services
// with asymmetric registration — for example, a task service that imports under
// fine-grained type strings (e.g., "http_read", "opc_scan") but exports under a single
// coarse type ("task").
func (s *Service) RegisterImporter(t string, i Importer) { s.importers[t] = i }

// RegisterExporter adds an Exporter for the given resource type. See RegisterImporter
// for the asymmetric-registration use case.
func (s *Service) RegisterExporter(e Exporter) { s.exporters[e.Type()] = e }

// ImporterType returns the ontology resource type that the importer registered
// under the given (possibly narrow) type string creates. For symmetric
// importers (e.g., "log") this is identical to the registration string; for
// asymmetric importers (e.g., a task service registered under "http_read")
// this is the broader ontology type ("task"). Returns an error if no importer
// is registered for t.
func (s *Service) ImporterType(t string) (ontology.ResourceType, error) {
	imp, ok := s.importers[t]
	if !ok {
		return "", errors.Newf("no importer registered for type %q", t)
	}
	return imp.Type(), nil
}

// Import validates and persists the given envelopes within a single transaction,
// returning the newly-assigned key for each resource in the same order as envs.
func (s *Service) Import(
	ctx context.Context,
	envelopes []Envelope,
) ([]string, error) {
	keys := make([]string, len(envelopes))
	if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		for i, env := range envelopes {
			importer, ok := s.importers[env.Type]
			if !ok {
				return errors.Newf("no importer registered for type %q", env.Type)
			}
			key, err := importer.Import(ctx, tx, env)
			if err != nil {
				return err
			}
			keys[i] = key
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// Export serializes the requested resources as envelopes. Each registered handler
// stamps its own per-schema version on the envelopes it returns.
func (s *Service) Export(
	ctx context.Context,
	resources []ontology.ID,
) ([]Envelope, error) {
	envelopes := make([]Envelope, len(resources))
	var err error
	for i, r := range resources {
		exporter, ok := s.exporters[r.Type]
		if !ok {
			return nil, errors.Newf(
				"no exporter registered for type %q",
				r.Type,
			)
		}
		if envelopes[i], err = exporter.Export(ctx, r.Key); err != nil {
			return nil, err
		}
	}
	return envelopes, nil
}
