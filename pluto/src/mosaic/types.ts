// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction } from "@synnaxlabs/x";
import { z } from "zod";

import { Tabs } from "@/tabs";

/**
 * Base interface for a mosaic node in the tree. Used to make sure that
 * zod type inference works correctly with recursive types.
 */
interface BaseNode {
  key: number;
  tabs?: Tabs.Tab[];
  selected?: string;
  direction?: direction.Direction;
  size?: number;
  first?: BaseNode;
  last?: BaseNode;
}

/**
 * Zod schema for a mosaic node. Used to validate the data for a node in the Mosaic
 * binary tree. See the `Node` interface for more information.
 */
export const nodeZ: z.ZodType<BaseNode> = z.object({
  key: z.number(),
  tabs: z.array(Tabs.tabZ).optional(),
  selected: z.string().optional(),
  direction: direction.directionZ.optional(),
  size: z.number().optional(),
  get first() {
    return nodeZ.optional();
  },
  get last() {
    return nodeZ.optional();
  },
});

/**
 * Represents the data for a node in the Mosaic binary tree. Nodes can be either leaf
 * or non-leaf nodes. Leaf nodes represent the edges of the tree, and render tabs in
 * the mosaic. Non-leaf nodes represent the internal nodes of the tree, and manage
 * the splitting/sizing of the tree.
 */
export interface Node {
  /**
   * Key assigns a unique identifier to the leaf. This key is used to identify the
   * location of the leaf in the tree. Keys are organized in a structure resembling
   * the following.
   *                   1
   *                 /   \
   *                 2    3
   *               /  \  / \
   *              4   5  6  7
   *
   * The first child of a node is assigned the key 2 * key, the second child is assigned
   * the key 2 * key + 1. The root node is assigned the key 1.
   */
  key: number;
  /**
   * A list of tabs for the node to render. If this value is defined, the node is
   * considered a leaf, and the tabs will be rendered.
   */
  tabs?: Tabs.Tab[];
  /** The key of the selected tab. This value only needs to be set for leaf nodes. */
  selected?: string;
  /** The direction of the split. This value only needs to be set for non-leaf nodes. */
  direction?: direction.Direction;
  /**
   * The first child of the node. If the node is vertical, this is the top child.
   * If the node is horizontal, this is the left child. This value only needs to be
   * set for non-leaf nodes.
   */
  first?: Node;
  /**
   * The last child of the node. If the node is vertical, this is the bottom child.
   * If the node is horizontal, this is the right child. This value only needs to be
   * set for non-leaf nodes.
   */
  last?: Node;
  /**
   * The size of the node as a decimal between 0 and 1. This value only needs to be
   * set for non-leaf nodes.
   */
  size?: number;
}
