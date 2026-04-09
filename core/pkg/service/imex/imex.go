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
	"github.com/synnaxlabs/x/gorp"
)

// Envelope is the standard JSON envelope for importing and exporting a single resource.
// The Version field identifies the schema version of the Data payload. The Type field
// identifies the ontology resource type and is used for routing to the correct service.
type Envelope struct {
	Version string         `json:"version" msgpack:"version"`
	Type    string         `json:"type" msgpack:"type"`
	Key     string         `json:"key" msgpack:"key"`
	Name    string         `json:"name" msgpack:"name"`
	Data    map[string]any `json:"data" msgpack:"data"`
}

// Importer can import a resource from an Envelope.
type Importer interface {
	Import(ctx context.Context, tx gorp.Tx, parent ontology.ID, env Envelope) error
}

// Exporter can export a resource to an Envelope.
type Exporter interface {
	Export(ctx context.Context, tx gorp.Tx, key string) (Envelope, error)
}

// ImporterExporter combines both interfaces.
type ImporterExporter interface {
	Importer
	Exporter
}
