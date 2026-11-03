// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"strings"

	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ gorp.Entry[Key] = Arc{}

// GorpKey implements gorp.Entry.
func (a Arc) GorpKey() Key { return a.Base().Key }

// SetOptions implements gorp.Entry.
func (Arc) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the Arc.
func (a Arc) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeArc, Key: a.Base().Key.String()}
}

// Type returns the type of the Arc's variant, or an empty type for an Arc without one.
func (a Arc) Type() Type {
	switch a.Variant.(type) {
	case AutomationArc:
		return TypeAutomation
	case ModuleArc:
		return TypeModule
	}
	return ""
}

// WithBase returns a copy of the Arc whose variant holds b. An Arc without a variant
// is returned unchanged.
func (a Arc) WithBase(b Base) Arc {
	switch v := a.Variant.(type) {
	case AutomationArc:
		v.Base = b
		a.Variant = v
	case ModuleArc:
		v.Base = b
		a.Variant = v
	}
	return a
}

// fileTypePrefix precedes the Arc's type in the type header of its file.
const fileTypePrefix = "arc-"

// FileType returns the type header of a file that holds an Arc of type t.
func FileType(t Type) string { return fileTypePrefix + string(t) }

// TypeFromFileType returns the Arc type a file's type header names. ok is false for a
// header that names no Arc type, such as the "arc" header of files before v4.
func TypeFromFileType(header string) (t Type, ok bool) {
	name, ok := strings.CutPrefix(header, fileTypePrefix)
	if !ok || !Type(name).IsValid() {
		return "", false
	}
	return Type(name), true
}

// exportBody is the Arc's portable body. The text is the materialized source rather
// than the operation log that reconstructs it, and the derived program is left off for
// the importer to recompile. The type is the file's type header.
type exportBody struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
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
	b := a.Base()
	return exportBody{
		Name:  b.Name,
		Type:  FileType(a.Type()),
		Mode:  b.Mode,
		Graph: b.Graph,
		Text:  exportText{Raw: b.Text.Materialize().Raw},
	}
}
