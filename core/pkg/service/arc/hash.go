// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"cmp"
	"encoding/json/v2"
	"fmt"
	"slices"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// semanticContent is the hash input: the fields that change an Arc's compiled behavior.
// Node positions and edge identities are excluded, so layout edits and a
// rebuilt-but-identical connection hash equally. Every field serializes, so the hash is
// a function of all of them and the inactive mode's fields contribute their zero.
type semanticContent struct {
	Mode      Mode                           `json:"mode"`
	Text      string                         `json:"text"`
	Functions ir.Functions                   `json:"functions"`
	Nodes     []string                       `json:"nodes"`
	Edges     []ir.Edge                      `json:"edges"`
	Inputs    map[string]msgpack.EncodedJSON `json:"inputs"`
}

// Hash returns the xxhash64 of the Arc's semantic content as 16 lowercase hex
// characters. Text hashes its materialized source, and a graph hashes without layout
// data. Equal content hashes equally regardless of edit history, so an edit that is
// undone restores the original hash. Text.Raw never contributes; the document is the
// source of truth.
func Hash(a Base) (string, error) {
	content := semanticContent{Mode: a.Source.Mode()}
	switch src := a.Source.Variant.(type) {
	case TextSource:
		content.Text = src.Text.Materialize().Raw
	case GraphSource:
		g := src.Graph
		content.Functions = g.Functions
		content.Nodes = make([]string, len(g.Nodes))
		for i, n := range g.Nodes {
			content.Nodes[i] = n.Key
		}
		slices.Sort(content.Nodes)
		content.Edges = make([]ir.Edge, len(g.Edges))
		for i, e := range g.Edges {
			content.Edges[i] = e.Edge
		}
		slices.SortFunc(content.Edges, compareEdges)
		content.Inputs = g.Inputs
	}
	b, err := json.Marshal(content, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrapf(err, "failed to hash arc %s", a.Key)
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}

// HashProgram returns the xxhash64 of a compiled program as 16 lowercase hex
// characters. The program holds the code of every module the Arc imports, so an
// edit to a module at any depth moves the hash of each importer.
func HashProgram(p arc.Program) (string, error) {
	b, err := json.Marshal(p, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash program")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}

// HashContent returns the compiled hash of an automation that does not compile, from
// the hashes of its content.
func HashContent(h drift.Hashes) (string, error) {
	b, err := json.Marshal(h, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash content")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}

func compareEdges(a, b ir.Edge) int {
	return cmp.Or(
		cmp.Compare(a.Source.Node, b.Source.Node),
		cmp.Compare(a.Source.Param, b.Source.Param),
		cmp.Compare(a.Target.Node, b.Target.Node),
		cmp.Compare(a.Target.Param, b.Target.Param),
		cmp.Compare(a.Kind, b.Kind),
	)
}
