// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeDirection } from "@synnaxlabs/x";

import { type Tabs } from "@/tabs";

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
   *              4  5  6  7
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
  direction?: CrudeDirection;
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
