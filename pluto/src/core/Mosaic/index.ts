// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Mosaic as CoreMosaic } from "./Mosaic";
import {
  insertMosaicTab,
  moveMosaicTab,
  removeMosaicTab,
  resizeMosaicNode,
  selectMosaicTab,
  renameMosaicTab,
  autoSelectTabs,
} from "./mosaicTree";
import { useMosaic } from "./useMosaic";
export * from "./types";

type CoreMosaicType = typeof CoreMosaic;

export interface MosaicType extends CoreMosaicType {
  /**
   * The Mosaic.use hook implements the state management and logic for the Mosaic
   * component. This hook should be used in conjunction with the Mosaic component.
   *
   * @param props - The props for the Mosaic.use hook.
   * @param props.initialValue - The initial root node of the mosaic tree. For
   * information on what this node should look like, see the {@link MosaicNode}
   * type.
   */
  use: typeof useMosaic;
  /**
   * Inserts a tab into a node in the mosaic. If the given key is not found,
   * the tab is inserted into the closest ancestor. This is to deal
   * with mosaic garbage collection.
   *
   * @param key - The key of the node to insert the tab into.
   * @param tab - The tab to insert.
   * @param loc - The location where the tab was 'dropped' relative to the node.
   */
  insertTab: typeof insertMosaicTab;
  /**
   * Removes a tab from the mosaic and performs any necessary garbage collection.
   *
   * @param root - The root of the mosaic.
   * @param tabKey - The key of the tab to remove. This tab must exist in the mosaic.
   */
  removeTab: typeof removeMosaicTab;
  /**
   * Marks the given tab as selected.
   *
   * @param root - The root of the mosaic.
   * @param tabKey - The key of the tab to select. This tab must exist in the mosaic.
   * @returns A shallow copy of the root of the mosaic with the tab selected.
   */
  selectTab: typeof selectMosaicTab;
  /**
   * Moves a tab from one node to another.
   *
   * @param root - The root of the mosaic.
   * @param to - The key of the node to move the tab to.
   * @param tabKey - The key of the tab to move. This tab must exist in the mosaic.
   * @param loc - The location where the tab was 'dropped' relative to the node.
   * @returns A shallow copy of the root of the mosaic with the tab moved.
   */
  moveTab: typeof moveMosaicTab;
  /**
   * Resizes the given mosaic leaf.
   *
   * @param root - The root of the mosaic.
   * @param key  - The key of the leaf to resize.
   * @param size - The new size distribution for the leaf. Expressed as either a percentage
   * or a number of pixels of the first child.
   * @returns A shallow copy of the root of the mosaic with the leaf resized.
   */
  resizeNode: typeof resizeMosaicNode;
  /**
   * Sets the title of a tab.
   *
   * @param root - The root of the mosaic.
   * @param tabKey  - The key of the tab to resize.
   * @param name - The new title of the tab.
   * @returns A shallow copy of the root of the mosaic with the tab title changed.
   */
  renameTab: typeof renameMosaicTab;
  /**
   * Automatically selects tabs for all nodes in the mosaic if they don't already have
   * a selection.
   *
   * @param root - The root of the mosaic.
   * @returns A shallow copy of the root with all nodes having a selection.
   */
  autoSelectTabs: typeof autoSelectTabs;
}

/***
 * Mosaic renders a tree of tab panes, with the ability to drag and drop tabs to
 * different locations in the tree as well as resize the panes (think of your typical
 * code editor). This component should be used in conjuction with the Mosaic.use hook
 * to implement the mosaic logic and maintain the state.
 *
 * @param props - The props for the Mosaic component. All props not listed below are
 * passed to the Tabs component of each set of tabs in the mosaic.
 * @param props.root - The root of the mosaic tree. This prop is provided by the
 *  Mosaic.use hook.
 * @param props.onDrop - The callback executed when a tab is dropped in a new location.
 * This prop is provided by the Mosaic.use hook.
 * @param props.onResize - The callback executed when a pane is resized. This prop is
 *  provided by the Mosaic.use hook.
 */
export const Mosaic = CoreMosaic as MosaicType;

Mosaic.use = useMosaic;
Mosaic.insertTab = insertMosaicTab;
Mosaic.removeTab = removeMosaicTab;
Mosaic.selectTab = selectMosaicTab;
Mosaic.moveTab = moveMosaicTab;
Mosaic.resizeNode = resizeMosaicNode;
Mosaic.renameTab = renameMosaicTab;
