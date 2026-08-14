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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
)

// bundleVersion is the version stamped on panel envelopes in project bundles.
const bundleVersion imex.Version = 0

// bundleBody is the envelope body for a panel bundle member.
type bundleBody struct {
	// Root is the panel tree with each resource reference rewritten to a bundle path.
	Root any `json:"root"`
}

// bundleTabResource is the bundle wire form of a resource tab: resource holds the
// target member's path from the bundle root instead of an ontology ID.
type bundleTabResource struct {
	TabBase
	Variant  TabType `json:"variant"`
	Resource string  `json:"resource"`
}

// bundleLeaf is the bundle wire form of a leaf node. Tabs holds a Tab for a view and a
// bundleTabResource for a resource.
type bundleLeaf struct {
	Variant NodeType `json:"variant"`
	Tabs    []any    `json:"tabs"`
}

// bundleSplit is the bundle wire form of a split node.
type bundleSplit struct {
	Variant   NodeType          `json:"variant"`
	Direction spatial.Direction `json:"direction"`
	Size      spatial.Decimal   `json:"size"`
	First     any               `json:"first"`
	Last      any               `json:"last"`
}

// EncodeBundle serializes p as a project-bundle member. Where the in-cluster tree holds
// an ontology.ID, the bundle tree holds the target member's path from the bundle root,
// taken from refs. A resource tab whose target refs does not name is removed, and
// leaves emptied by removals are collapsed. View tabs pass through unchanged. It
// returns a validation error when p has no name.
func EncodeBundle(p Panel, refs map[ontology.ID]string) (imex.Envelope, error) {
	root := stripNonMemberTabs(p.Root, refs)
	collapseEmptyLeaves(&root)
	wire, err := bundleNode(root, refs)
	if err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: bundleVersion,
		Type:    string(ontology.ResourceTypePanel),
		Name:    p.Name,
	}
	if err = imex.Encode(&env, bundleBody{Root: wire}); err != nil {
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

// bundleNode converts a node to its bundle wire form, rewriting each resource tab's
// target to its path from refs. n is a tree stripNonMemberTabs already filtered, so a
// target absent from refs is a programmer error, not bad input.
func bundleNode(n Node, refs map[ontology.ID]string) (any, error) {
	switch v := n.Variant.(type) {
	case NodeLeaf:
		tabs := make([]any, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			r, ok := t.Variant.(TabResource)
			if !ok {
				tabs = append(tabs, t)
				continue
			}
			path, ok := refs[r.Resource]
			if !ok {
				return nil, errors.Newf("no bundle path for resource %s", r.Resource)
			}
			tabs = append(tabs, bundleTabResource{
				TabBase:  r.TabBase,
				Variant:  TabTypeResource,
				Resource: path,
			})
		}
		return bundleLeaf{Variant: NodeTypeLeaf, Tabs: tabs}, nil
	case NodeSplit:
		first, err := bundleNode(v.First, refs)
		if err != nil {
			return nil, err
		}
		last, err := bundleNode(v.Last, refs)
		if err != nil {
			return nil, err
		}
		return bundleSplit{
			Variant:   NodeTypeSplit,
			Direction: v.Direction,
			Size:      v.Size,
			First:     first,
			Last:      last,
		}, nil
	default:
		return nil, nil
	}
}
