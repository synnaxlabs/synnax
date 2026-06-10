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
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Service is the central import/export registry. Handlers are registered via
// RegisterImportExporter, RegisterImporter, or RegisterExporter, and Import and Export
// route to them by Type. Service is safe for concurrent registration and lookup.
type Service struct {
	mu        sync.RWMutex
	importers map[string]Importer
	exporters map[ontology.ResourceType]Exporter
}

// NewService creates a new, empty import/export registry. Handlers register themselves
// via RegisterImportExporter, RegisterImporter, or RegisterExporter at their own
// startup time, typically by accepting the Service in their own service config.
func NewService() *Service {
	return &Service{
		importers: make(map[string]Importer),
		exporters: make(map[ontology.ResourceType]Exporter),
	}
}

// RegisterImportExporter registers a single handler for both halves under its own Type.
// Use this for symmetric services (one resource type for both import and export) —
// e.g., logs, schematics.
func (s *Service) RegisterImportExporter(ie ImportExporter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := ie.Type()
	s.importers[string(t)] = ie
	s.exporters[t] = ie
}

// RegisterImporter adds an [Importer] for the given resource type. Use this for
// services with asymmetric registration — for example, a task service that imports
// under fine-grained type strings (e.g., "http_read", "opc_scan") but exports under a
// single coarse type ("task").
func (s *Service) RegisterImporter(t string, i Importer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.importers[t] = i
}

// RegisterExporter adds an [Exporter] for the given resource type. See
// [Service.RegisterImporter] for the asymmetric-registration use case.
func (s *Service) RegisterExporter(e Exporter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exporters[e.Type()] = e
}

// ImporterType returns the ontology resource type that the importer registered under
// the given (possibly narrow) type string creates. For symmetric importers (e.g.,
// "log") this is identical to the registration string; for asymmetric importers (e.g.,
// a task service registered under "http_read") this is the broader ontology type
// ("task"). Returns a validation error scoped to the "type" field if no importer is
// registered for t.
func (s *Service) ImporterType(t string) (ontology.ResourceType, error) {
	s.mu.RLock()
	imp, ok := s.importers[t]
	s.mu.RUnlock()
	if !ok {
		return "", notFoundError(t, "type", "importer")
	}
	return imp.Type(), nil
}

// notFoundError builds the validation error returned when Import or Export is called
// with a Type that has no registered handler. kind is "importer" or "exporter"; path is
// the JSON path the API layer surfaces the error against (always "type" today).
func notFoundError(t any, path, kind string) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "no %s registered for type %q", kind, t),
		path,
	)
}

// Import routes envelope to the importer registered under envelope.Type, persists it on
// tx, and returns the newly-assigned key. Returns a validation error scoped to the
// "type" field if no importer is registered for envelope.Type. Callers that need
// multi-envelope atomicity should wrap several Import calls in db.WithTx.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	envelope Envelope,
) (Key, error) {
	s.mu.RLock()
	importer, ok := s.importers[envelope.Type]
	s.mu.RUnlock()
	if !ok {
		return "", notFoundError(envelope.Type, "type", "importer")
	}
	key, err := importer.Import(ctx, tx, envelope)
	if err != nil {
		return "", errors.Wrap(err, "failed to import envelope")
	}
	return key, nil
}

// Export routes resource to the exporter registered under resource.Type, retrieves it
// on tx, and returns the resulting envelope. The exporter stamps its own per-schema
// version on the envelope. Returns a validation error scoped to the "type" field if no
// exporter is registered for resource.Type.
func (s *Service) Export(
	ctx context.Context,
	tx gorp.Tx,
	resource ontology.ID,
) (Envelope, error) {
	s.mu.RLock()
	exporter, ok := s.exporters[resource.Type]
	s.mu.RUnlock()
	if !ok {
		return Envelope{}, notFoundError(resource.Type, "type", "exporter")
	}
	env, err := exporter.Export(ctx, tx, resource.Key)
	if err != nil {
		return Envelope{}, errors.Wrap(err, "failed to export resource")
	}
	return env, nil
}
