// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { CrudeLocation } from "@synnaxlabs/x";

import {
  moveMosaicTab,
  removeMosaicTab,
  resizeMosaicNode,
  selectMosaicTab,
  renameMosaicTab,
  autoSelectTabs,
} from "./mosaicTree";
import { MosaicNode } from "./types";

export interface UseMosaicProps {
  allowRename?: boolean;
  initialTree: MosaicNode;
}

export interface UseMosaicReturn {
  root: MosaicNode;
  onDrop: (key: number, tabKey: string, loc: CrudeLocation) => void;
  onClose: (tabKey: string) => void;
  onSelect: (tabKey: string) => void;
  onResize: (key: number, sizes: number) => void;
  onRename?: (tabKey: string, name: string) => void;
}

export const useMosaic = ({
  allowRename = false,
  initialTree,
}: UseMosaicProps): UseMosaicReturn => {
  const [root, setRoot] = useState(initialTree);

  useEffect(() => setRoot(autoSelectTabs(initialTree)[0]), [initialTree]);

  const handleDrop = (key: number, tabKey: string, loc: CrudeLocation): void =>
    setRoot((r) => moveMosaicTab(r, tabKey, loc, key)[0]);

  const handleClose = (tabKey: string): void =>
    setRoot((r) => removeMosaicTab(r, tabKey)[0]);

  const handleSelect = (tabKey: string): void =>
    setRoot((r) => selectMosaicTab(r, tabKey));

  const handleResize = (key: number, size: number): void =>
    setRoot((r) => resizeMosaicNode(r, key, size));

  const handleRename = (tabKey: string, title: string): void =>
    setRoot((r) => renameMosaicTab(r, tabKey, title));

  return {
    root,
    onDrop: handleDrop,
    onClose: handleClose,
    onSelect: handleSelect,
    onResize: handleResize,
    onRename: allowRename ? handleRename : undefined,
  };
};
