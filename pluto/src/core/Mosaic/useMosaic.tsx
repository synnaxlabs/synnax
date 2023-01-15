// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import {
  moveMosaicTab,
  removeMosaicTab,
  resizeMosaicLeaf,
  selectMosaicTab,
  renameMosaicTab,
  autoSelectTabs,
} from "./mosaicTree";
import { MosaicLeaf } from "./types";

import { Location } from "@/spatial";

export interface UseMosaicProps {
  allowRename?: boolean;
  initialTree: MosaicLeaf;
}

export interface UseMosaicReturn {
  root: MosaicLeaf;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
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

  useEffect(() => {
    setRoot(autoSelectTabs(initialTree));
  }, [initialTree]);

  const handleDrop = (key: number, tabKey: string, loc: Location): void =>
    setRoot((r) => moveMosaicTab(r, tabKey, loc, key));

  const handleClose = (tabKey: string): void =>
    setRoot((r) => removeMosaicTab(r, tabKey));

  const handleSelect = (tabKey: string): void =>
    setRoot((r) => selectMosaicTab(r, tabKey));

  const handleResized = (key: number, size: number): void =>
    setRoot((r) => resizeMosaicLeaf(r, key, size));

  const handleRename = (tabKey: string, title: string): void =>
    setRoot((r) => renameMosaicTab(r, tabKey, title));

  return {
    root,
    onDrop: handleDrop,
    onClose: handleClose,
    onSelect: handleSelect,
    onResize: handleResized,
    onRename: allowRename ? handleRename : undefined,
  };
};
