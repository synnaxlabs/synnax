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
