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
	"encoding/json/v2"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/color"
	colorv1 "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// bundleBody is the envelope body for a panel bundle member. C is the color type of
// the member's version.
type bundleBody[C any] struct {
	// Root is the panel tree with each resource reference rewritten to a bundle path.
	Root  any     `json:"root"`
	Icon  *string `json:"icon,omitzero"`
	Color *C      `json:"color,omitzero"`
}

// bundleResourceTab is the bundle wire form of a resource tab: resource holds the
// target member's path from the bundle root instead of a resource ID.
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
// a resource.ID, the bundle tree holds the target member's path from the bundle root,
// taken from refs. A resource tab whose target refs does not name is removed, and
// leaves emptied by removals are collapsed. View tabs pass through unchanged. It
// returns a validation error when p has no name.
func EncodeBundle(p Panel, refs map[resource.ID]string) (imex.Envelope, error) {
	root := stripTabs(p.Root, func(id resource.ID) bool {
		_, member := refs[id]
		return !member
	})
	collapseEmptyLeaves(&root)
	node, err := bundleNode(root, refs)
	if err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest,
		Type:    string(resource.TypePanel),
		Name:    p.Name,
	}
	body := bundleBody[color.Color]{Root: node, Icon: p.Icon, Color: p.Color}
	if err := env.Encode(body); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// DecodeBundle deserializes a project-bundle member produced by EncodeBundle. Where the
// bundle tree holds a member path, the returned tree holds the resource.ID refs maps
// that path to; a path refs does not hold is a validation error naming it. A version
// above versions.Latest is a path-scoped validation error. Arc explorer tabs are
// removed. The returned panel carries no key or project; the caller assigns both.
func DecodeBundle(
	ctx context.Context,
	env imex.Envelope,
	refs map[string]resource.ID,
) (Panel, error) {
	if env.Version > versions.Latest {
		return Panel{}, imex.NewErrUnsupportedVersion(
			string(resource.TypePanel), env.Version, versions.Latest,
		)
	}
	body, err := decodeBundleBody(ctx, env)
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
	v3.StripArcExplorerTabs(&root)
	return Panel{Name: env.Name, Root: root, Icon: body.Icon, Color: body.Color}, nil
}

// decodeBundleBody decodes the body of env. A member from before panel v4 holds a color
// with a 0-1 alpha, which it migrates to a byte alpha.
func decodeBundleBody(
	ctx context.Context,
	env imex.Envelope,
) (bundleBody[color.Color], error) {
	if env.Version >= v4.Version {
		return env.Decode[bundleBody[color.Color]](ctx)
	}
	old, err := env.Decode[bundleBody[colorv1.Color]](ctx)
	if err != nil || old.Color == nil {
		return bundleBody[color.Color]{Root: old.Root, Icon: old.Icon}, err
	}
	c, err := colorv2.MigrateColor(ctx, *old.Color)
	return bundleBody[color.Color]{Root: old.Root, Icon: old.Icon, Color: &c}, err
}

// resolveBundleNode rewrites each resource tab's bundle path to the resource.ID refs
// maps it to, returning the tree in the in-cluster wire form. A node the tree grammar
// cannot hold is left for Node's decoder to reject.
func resolveBundleNode(node any, refs map[string]resource.ID) (any, error) {
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
func ResourceRefs(root Node) []resource.ID {
	ids := set.New[resource.ID]()
	collectResourceRefs(root, ids)
	return lo.Keys(ids)
}

// collectResourceRefs adds the ID of every resource the tree's resource tabs reference
// to ids.
func collectResourceRefs(n Node, ids set.Set[resource.ID]) {
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

// stripTabs removes every resource tab whose resource dropped reports, leaving emptied
// leaves in place for collapseEmptyLeaves.
func stripTabs(n Node, dropped func(resource.ID) bool) Node {
	switch v := n.Variant.(type) {
	case LeafNode:
		v.Tabs = slices.DeleteFunc(slices.Clone(v.Tabs), func(t Tab) bool {
			r, ok := t.Variant.(ResourceTab)
			return ok && dropped(r.Resource)
		})
		return Node{Variant: v}
	case SplitNode:
		v.First = stripTabs(v.First, dropped)
		v.Last = stripTabs(v.Last, dropped)
		return Node{Variant: v}
	default:
		return n
	}
}

// bundleNode converts a node to its bundle wire form, rewriting each resource tab's
// target to its path from refs. stripTabs runs first, so a tab whose target refs does
// not name is an invariant violation and errors.
func bundleNode(n Node, refs map[resource.ID]string) (any, error) {
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
