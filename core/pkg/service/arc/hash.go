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
	"encoding/json"
	"fmt"
	"slices"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// semanticContent is the hash input: the fields that change an arc's compiled
// behavior. Node positions and edge identities are excluded, so layout edits and a
// rebuilt-but-identical connection hash equally.
type semanticContent struct {
	Mode      Mode                           `json:"mode"`
	Text      string                         `json:"text,omitempty"`
	Functions ir.Functions                   `json:"functions,omitempty"`
	Nodes     []string                       `json:"nodes,omitempty"`
	Edges     []ir.Edge                      `json:"edges,omitempty"`
	Inputs    map[string]msgpack.EncodedJSON `json:"inputs,omitempty"`
}

// Hash returns the xxhash64 of the arc's semantic content as 16 lowercase hex
// characters. Only the active mode's source contributes: text mode hashes the
// materialized source, graph mode hashes the graph without layout data. Equal content
// hashes equally regardless of edit history, so an edit that is undone restores the
// original hash. A non-empty Text.Raw is trusted as the materialized source; callers
// holding one must not have mutated the document since it was derived.
func Hash(a Arc) (string, error) {
	content := semanticContent{Mode: a.Mode}
	if a.Mode == ModeText {
		content.Text = a.Text.Raw
		if content.Text == "" {
			content.Text = a.Text.Materialize().Raw
		}
	} else {
		content.Functions = a.Graph.Functions
		content.Nodes = make([]string, len(a.Graph.Nodes))
		for i, n := range a.Graph.Nodes {
			content.Nodes[i] = n.Key
		}
		slices.Sort(content.Nodes)
		content.Edges = make([]ir.Edge, len(a.Graph.Edges))
		for i, e := range a.Graph.Edges {
			content.Edges[i] = e.Edge
		}
		slices.SortFunc(content.Edges, compareEdges)
		content.Inputs = a.Graph.Inputs
	}
	b, err := json.Marshal(content)
	if err != nil {
		return "", errors.Wrapf(err, "failed to hash arc %s", a.Key)
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
