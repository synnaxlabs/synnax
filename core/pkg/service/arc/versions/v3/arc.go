// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Arc{}

// GorpKey implements gorp.Entry.
func (a Arc) GorpKey() Key { return a.Key }

// SetOptions implements gorp.Entry.
func (Arc) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the Arc.
func (a Arc) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeArc, Key: a.Key.String()}
}

// exportBody is the Arc's portable body. The text is the materialized source rather
// than the operation log that reconstructs it, and the derived program and status are
// left off: an importer recompiles the one and the runtime owns the other.
type exportBody struct {
	Name  string      `json:"name"`
	Mode  Mode        `json:"mode"`
	Graph graph.Graph `json:"graph"`
	Text  exportText  `json:"text"`
}

// exportText is an Arc's text as a file carries it. Raw is the field Import reads, and
// the writer seeds a fresh operation log from it.
type exportText struct {
	Raw string `json:"raw"`
}

var _ imex.BodyExporter = Arc{}

// ExportBody implements imex.BodyExporter.
func (a Arc) ExportBody() any {
	return exportBody{
		Name:  a.Name,
		Mode:  a.Mode,
		Graph: a.Graph,
		Text:  exportText{Raw: a.Text.Materialize().Raw},
	}
}
