// Copyright 2022 Synnax Labs, Inc.
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

import { Location } from "@/util/spatial";

export interface UseMosaicProps {
  editableTitle?: boolean;
  initialTree: MosaicLeaf;
}

export interface UseMosaicReturn {
  root: MosaicLeaf;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onClose: (tabKey: string) => void;
  onSelect: (tabKey: string) => void;
  onResize: (key: number, sizes: number) => void;
  onTitleChange?: (tabKey: string, title: string) => void;
}

export const useMosaic = ({
  editableTitle = false,
  initialTree,
}: UseMosaicProps): UseMosaicReturn => {
  const [root, setRoot] = useState(initialTree);

  useEffect(() => {
    setRoot(autoSelectTabs(initialTree));
  }, [initialTree]);

  const onDrop = (key: number, tabKey: string, loc: Location): void =>
    setRoot((r) => moveMosaicTab(r, tabKey, loc, key));

  const onClose = (tabKey: string): void => setRoot((r) => removeMosaicTab(r, tabKey));

  const onSelect = (tabKey: string): void => setRoot((r) => selectMosaicTab(r, tabKey));

  const onResize = (key: number, size: number): void =>
    setRoot((r) => resizeMosaicLeaf(r, key, size));

  const onTitleChange = (tabKey: string, title: string): void => {
    setRoot((r) => renameMosaicTab(r, tabKey, title));
  };

  return {
    root,
    onDrop,
    onClose,
    onSelect,
    onResize,
    onTitleChange: editableTitle ? onTitleChange : undefined,
  };
};
