// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

import {
  autoSelectTabs,
  moveTab,
  removeTab,
  renameTab,
  resizeNode,
  selectTab,
} from "@/mosaic/tree";
import { type Node } from "@/mosaic/types";

export interface UseProps {
  allowRename?: boolean;
  initialTree: Node;
}

export interface UseReturn {
  root: Node;
  onDrop: (key: number, tabKey: string, loc: location.Location) => void;
  onClose: (tabKey: string) => void;
  onSelect: (tabKey: string) => void;
  onResize: (key: number, sizes: number) => void;
  onRename?: (tabKey: string, name: string) => void;
}

/**
 * The Mosaic.use hook implements the state management and logic for the Mosaic
 * component. This hook should be used in conjunction with the Mosaic component.
 *
 * @param props - The props for the Mosaic.use hook.
 * @param props.initialValue - The initial root node of the mosaic tree. For
 * information on what this node should look like, see the {@link Node}
 * type.
 */
export const use = ({ allowRename = false, initialTree }: UseProps): UseReturn => {
  const [root, setRoot] = useState(initialTree);

  useEffect(() => setRoot(autoSelectTabs(initialTree)[0]), [initialTree]);

  const handleDrop = (key: number, tabKey: string, loc: location.Location): void =>
    setRoot((r) => moveTab(r, tabKey, loc, key)[0]);

  const handleClose = (tabKey: string): void => setRoot((r) => removeTab(r, tabKey)[0]);

  const handleSelect = (tabKey: string): void => setRoot((r) => selectTab(r, tabKey));

  const handleResize = (key: number, size: number): void =>
    setRoot((r) => resizeNode(r, key, size));

  const handleRename = (tabKey: string, title: string): void =>
    setRoot((r) => renameTab(r, tabKey, title));

  return {
    root,
    onDrop: handleDrop,
    onClose: handleClose,
    onSelect: handleSelect,
    onResize: handleResize,
    onRename: allowRename ? handleRename : undefined,
  };
};
