// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
)

// bundleVersion is the version stamped on panel envelopes in project bundles.
const bundleVersion imex.Version = 0

// bundleBody is the envelope body for a panel bundle member.
type bundleBody struct {
	// Root is the panel tree with each resource reference rewritten to a bundle path.
	Root map[string]any `json:"root"`
}

// EncodeBundle serializes p as a project-bundle member. Where the in-cluster tree
// holds an ontology.ID, the bundle tree holds the target member's path from the
// bundle root, taken from refs. A resource tab whose target refs does not name is
// removed, and leaves emptied by removals are collapsed. View tabs pass through
// unchanged. It returns a validation error when p has no name.
func EncodeBundle(p Panel, refs map[ontology.ID]string) (imex.Envelope, error) {
	root := stripNonMemberTabs(p.Root, refs)
	collapseEmptyLeaves(&root)
	raw, err := json.Marshal(root)
	if err != nil {
		return imex.Envelope{}, err
	}
	var m map[string]any
	if err = json.Unmarshal(raw, &m); err != nil {
		return imex.Envelope{}, err
	}
	if err = rewriteResourceTabs(m, refs); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: bundleVersion,
		Type:    string(ontology.ResourceTypePanel),
		Name:    p.Name,
	}
	if err = imex.Encode(&env, bundleBody{Root: m}); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// stripNonMemberTabs removes every resource tab whose target refs does not name,
// leaving emptied leaves in place for collapseEmptyLeaves.
func stripNonMemberTabs(n Node, refs map[ontology.ID]string) Node {
	switch v := n.Variant.(type) {
	case NodeLeaf:
		tabs := make([]Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if r, ok := t.Variant.(TabResource); ok {
				if _, member := refs[r.Resource]; !member {
					continue
				}
			}
			tabs = append(tabs, t)
		}
		v.Tabs = tabs
		return Node{Variant: v}
	case NodeSplit:
		v.First = stripNonMemberTabs(v.First, refs)
		v.Last = stripNonMemberTabs(v.Last, refs)
		return Node{Variant: v}
	default:
		return n
	}
}

// rewriteResourceTabs replaces each resource tab's {type, key} object with the
// target's bundle path. node is the JSON form of a tree stripNonMemberTabs already
// filtered, so a target absent from refs is a programmer error, not bad input.
func rewriteResourceTabs(node map[string]any, refs map[ontology.ID]string) error {
	if node["variant"] == string(NodeTypeLeaf) {
		tabs, _ := node["tabs"].([]any)
		for _, t := range tabs {
			tab, ok := t.(map[string]any)
			if !ok || tab["variant"] != string(TabTypeResource) {
				continue
			}
			resource, ok := tab["resource"].(map[string]any)
			if !ok {
				return errors.Newf("resource tab holds no resource object")
			}
			typ, _ := resource["type"].(string)
			key, _ := resource["key"].(string)
			id := ontology.ID{Type: ontology.ResourceType(typ), Key: key}
			path, ok := refs[id]
			if !ok {
				return errors.Newf("no bundle path for resource %s", id)
			}
			tab["resource"] = path
		}
		return nil
	}
	for _, side := range []string{"first", "last"} {
		if child, ok := node[side].(map[string]any); ok {
			if err := rewriteResourceTabs(child, refs); err != nil {
				return err
			}
		}
	}
	return nil
}
