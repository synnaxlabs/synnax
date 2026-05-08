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
	"encoding/json"

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
	// [REQUIRED]
	DB *gorp.DB
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening an
	// import/export service.
	DefaultServiceConfig = ServiceConfig{}
)

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
// ServiceConfig.ImporterExporters at open time and routed by their Type.
type Service struct {
	cfg       ServiceConfig
	importers map[ontology.ResourceType]Importer
	exporters map[ontology.ResourceType]Exporter
}

// OpenService creates a new, empty import/export registry. Handlers register
// themselves via RegisterImporterExporter / RegisterImporter / RegisterExporter
// at their own open time, typically by accepting the imex Service in their own
// service config. Each handler is responsible for stamping its own per-schema
// version on the envelopes it produces; the registry does not impose a
// centralized version.
func OpenService(_ context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:       cfg,
		importers: make(map[ontology.ResourceType]Importer),
		exporters: make(map[ontology.ResourceType]Exporter),
	}, nil
}

// RegisterImporterExporter registers a single handler for both halves under
// its own Type. Use this for symmetric services (one resource type for both
// import and export) — e.g., logs, schematics.
func (s *Service) RegisterImporterExporter(ie ImporterExporter) {
	t := ie.Type()
	s.importers[t] = ie
	s.exporters[t] = ie
}

// RegisterImporter adds an Importer for the given resource type. Use this for
// services with asymmetric registration — for example, a task service that
// imports under fine-grained type strings (e.g., "http_read", "opc_scan") but
// exports under a single coarse type ("task").
func (s *Service) RegisterImporter(t ontology.ResourceType, i Importer) {
	s.importers[t] = i
}

// RegisterExporter adds an Exporter for the given resource type. See
// RegisterImporter for the asymmetric-registration use case.
func (s *Service) RegisterExporter(t ontology.ResourceType, e Exporter) {
	s.exporters[t] = e
}

// Import validates and persists the given envelopes within a single
// transaction, returning the newly-assigned key for each resource in the same
// order as envs. The envelope's key is ignored; each handler always assigns a
// fresh key.
func (s *Service) Import(
	ctx context.Context,
	envs []Envelope,
) ([]string, error) {
	keys := make([]string, len(envs))
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		for i, env := range envs {
			imp, ok := s.importers[ontology.ResourceType(env.Type)]
			if !ok {
				return errors.Newf(
					"no importer registered for type %q",
					env.Type,
				)
			}
			payload, err := decodeImportPayload(env)
			if err != nil {
				return err
			}
			key, err := imp.Import(ctx, tx, payload)
			if err != nil {
				return err
			}
			keys[i] = key
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Export serializes the requested resources as envelopes. Each registered
// handler stamps its own per-schema version on the envelopes it returns.
func (s *Service) Export(
	ctx context.Context,
	resources []ontology.ID,
) ([]Envelope, error) {
	result := make([]Envelope, 0, len(resources))
	for _, r := range resources {
		exp, ok := s.exporters[r.Type]
		if !ok {
			return nil, errors.Newf(
				"no exporter registered for type %q",
				r.Type,
			)
		}
		env, err := exp.Export(ctx, r.Key)
		if err != nil {
			return nil, err
		}
		result = append(result, env)
	}
	return result, nil
}

// decodeImportPayload decodes the envelope's Data into a generic
// map[string]any so the handler can route on Version and dispatch to the right
// schema parser. The envelope's promoted Key is intentionally dropped.
func decodeImportPayload(env Envelope) (ImportPayload, error) {
	payload := ImportPayload{Version: env.Version, Name: env.Name}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &payload.Data); err != nil {
			return ImportPayload{}, errors.Wrap(err, "envelope data must be a JSON object")
		}
	}
	return payload, nil
}
