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
	"context"
	"encoding/json"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// bundleBody is the envelope body for a panel bundle member.
type bundleBody struct {
	// Root is the panel tree with each resource reference rewritten to a bundle path.
	Root any `json:"root"`
}

// bundleResourceTab is the bundle wire form of a resource tab: resource holds the
// target member's path from the bundle root instead of an ontology ID.
type bundleResourceTab struct {
	TabBase
	Variant  TabType `json:"variant"`
	Resource string  `json:"resource"`
}

// bundleLeaf is the bundle wire form of a leaf node. Tabs holds a Tab for a view and a
// bundleResourceTab for a resource.
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
	node, err := bundleNode(root, refs)
	if err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest,
		Type:    string(ontology.ResourceTypePanel),
		Name:    p.Name,
	}
	if err := imex.Encode(&env, bundleBody{Root: node}); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// DecodeBundle deserializes a project-bundle member produced by EncodeBundle. Where the
// bundle tree holds a member path, the returned tree holds the ontology.ID refs maps
// that path to; a path refs does not hold is a validation error naming it. A version
// above versions.Latest is a path-scoped validation error. The returned panel carries
// no key or parent; the caller assigns both.
func DecodeBundle(
	ctx context.Context,
	env imex.Envelope,
	refs map[string]ontology.ID,
) (Panel, error) {
	if env.Version > versions.Latest {
		return Panel{}, imex.NewErrUnsupportedVersion(
			string(ontology.ResourceTypePanel), env.Version, versions.Latest,
		)
	}
	body, err := imex.Decode[bundleBody](ctx, env)
	if err != nil {
		return Panel{}, err
	}
	resolved, err := resolveBundleNode(body.Root, refs)
	if err != nil {
		return Panel{}, err
	}
	data, err := json.Marshal(resolved)
	if err != nil {
		return Panel{}, err
	}
	var root Node
	if err := json.Unmarshal(data, &root); err != nil {
		return Panel{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return Panel{Name: env.Name, Root: root}, nil
}

// resolveBundleNode rewrites each resource tab's bundle path to the ontology.ID refs
// maps it to, returning the tree in the in-cluster wire form. A node the tree grammar
// cannot hold is left for Node's decoder to reject.
func resolveBundleNode(node any, refs map[string]ontology.ID) (any, error) {
	m, ok := node.(map[string]any)
	if !ok {
		return node, nil
	}
	switch m["variant"] {
	case string(LeafNodeType):
		tabs, _ := m["tabs"].([]any)
		for _, rawTab := range tabs {
			tab, ok := rawTab.(map[string]any)
			if !ok || tab["variant"] != string(ResourceTabType) {
				continue
			}
			path, ok := tab["resource"].(string)
			if !ok {
				return nil, errors.Wrap(
					validate.ErrValidation,
					"resource tab holds no member path",
				)
			}
			id, ok := refs[path]
			if !ok {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"references %q, which is not a member of the bundle", path,
				)
			}
			tab["resource"] = id
		}
	case string(SplitNodeType):
		for _, side := range []string{"first", "last"} {
			resolved, err := resolveBundleNode(m[side], refs)
			if err != nil {
				return nil, err
			}
			m[side] = resolved
		}
	}
	return m, nil
}

// ResourceRefs returns the ID of every resource the tree's resource tabs reference,
// each once. Order is unspecified.
func ResourceRefs(root Node) []ontology.ID {
	ids := set.New[ontology.ID]()
	collectResourceRefs(root, ids)
	return lo.Keys(ids)
}

// collectResourceRefs adds the ID of every resource the tree's resource tabs reference
// to ids.
func collectResourceRefs(n Node, ids set.Set[ontology.ID]) {
	switch v := n.Variant.(type) {
	case LeafNode:
		for _, t := range v.Tabs {
			if r, ok := t.Variant.(ResourceTab); ok {
				ids.Add(r.Resource)
			}
		}
	case SplitNode:
		collectResourceRefs(v.First, ids)
		collectResourceRefs(v.Last, ids)
	}
}

// stripNonMemberTabs removes every resource tab whose target refs does not name,
// leaving emptied leaves in place for collapseEmptyLeaves.
func stripNonMemberTabs(n Node, refs map[ontology.ID]string) Node {
	switch v := n.Variant.(type) {
	case LeafNode:
		tabs := make([]Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if r, ok := t.Variant.(ResourceTab); ok {
				if _, member := refs[r.Resource]; !member {
					continue
				}
			}
			tabs = append(tabs, t)
		}
		v.Tabs = tabs
		return Node{Variant: v}
	case SplitNode:
		v.First = stripNonMemberTabs(v.First, refs)
		v.Last = stripNonMemberTabs(v.Last, refs)
		return Node{Variant: v}
	default:
		return n
	}
}

// bundleNode converts a node to its bundle wire form, rewriting each resource tab's
// target to its path from refs. stripNonMemberTabs runs first, so a tab whose target
// refs does not name is an invariant violation and errors.
func bundleNode(n Node, refs map[ontology.ID]string) (any, error) {
	switch v := n.Variant.(type) {
	case LeafNode:
		tabs := make([]any, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			r, ok := t.Variant.(ResourceTab)
			if !ok {
				tabs = append(tabs, t)
				continue
			}
			path, ok := refs[r.Resource]
			if !ok {
				return nil, errors.Newf(
					"panel tab references non-member resource %s", r.Resource,
				)
			}
			tabs = append(tabs, bundleResourceTab{
				TabBase:  r.TabBase,
				Variant:  ResourceTabType,
				Resource: path,
			})
		}
		return bundleLeaf{Variant: LeafNodeType, Tabs: tabs}, nil
	case SplitNode:
		first, err := bundleNode(v.First, refs)
		if err != nil {
			return nil, err
		}
		last, err := bundleNode(v.Last, refs)
		if err != nil {
			return nil, err
		}
		return bundleSplit{
			Variant:   SplitNodeType,
			Direction: v.Direction,
			Size:      v.Size,
			First:     first,
			Last:      last,
		}, nil
	default:
		return nil, nil
	}
}
