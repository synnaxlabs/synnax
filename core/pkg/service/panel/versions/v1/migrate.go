// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	"github.com/synnaxlabs/x/errors"
)

// MigrateNode lifts a v0 node to v1. The stored bytes are unchanged: v1 only widens
// the resource types a tab may display.
func MigrateNode(ctx context.Context, old v0.Node) (Node, error) {
	switch v := old.Variant.(type) {
	case v0.LeafNode:
		tabs := make([]Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			tab, err := MigrateTab(ctx, t)
			if err != nil {
				return Node{}, err
			}
			tabs = append(tabs, tab)
		}
		return Node{Variant: LeafNode{Tabs: tabs}}, nil
	case v0.SplitNode:
		first, err := MigrateNode(ctx, v.First)
		if err != nil {
			return Node{}, err
		}
		last, err := MigrateNode(ctx, v.Last)
		if err != nil {
			return Node{}, err
		}
		return Node{Variant: SplitNode{
			Direction: v.Direction,
			Size:      v.Size,
			First:     first,
			Last:      last,
		}}, nil
	}
	return Node{}, errors.Newf("unknown node variant %T", old.Variant)
}

// MigrateTab lifts a v0 tab to v1.
func MigrateTab(ctx context.Context, old v0.Tab) (Tab, error) {
	switch v := old.Variant.(type) {
	case v0.ResourceTab:
		resource, err := ontology.MigrateID(ctx, v.Resource)
		if err != nil {
			return Tab{}, err
		}
		return Tab{Variant: ResourceTab{TabBase: v.TabBase, Resource: resource}}, nil
	case v0.ViewTab:
		return Tab{Variant: ViewTab{TabBase: v.TabBase, View: v.View}}, nil
	}
	return Tab{}, errors.Newf("unknown tab variant %T", old.Variant)
}
