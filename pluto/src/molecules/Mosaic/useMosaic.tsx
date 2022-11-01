import { useState } from "react";
import {
  moveMosaicTab,
  removeMosaicTab,
  resizeMosaicLeaf,
  selectMosaicTab,
} from "./mosaicTree";
import { MosaicLeaf } from "./types";
import { Location } from "@/util";

export interface UseMosaicProps {
  initialTree: MosaicLeaf;
}

export interface UseMosaicReturn {
  root: MosaicLeaf;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onClose: (tabKey: string) => void;
  onSelect: (tabKey: string) => void;
  onResize: (key: number, sizes: number) => void;
}

export const useMosaic = ({ initialTree }: UseMosaicProps): UseMosaicReturn => {
  const [root, setRoot] = useState(initialTree);

  const onDrop = (key: number, tabKey: string, loc: Location) =>
    setRoot((r) => moveMosaicTab(r, tabKey, loc, key));

  const onClose = (tabKey: string) =>
    setRoot((r) => removeMosaicTab(r, tabKey));

  const onSelect = (tabKey: string) =>
    setRoot((r) => selectMosaicTab(r, tabKey));

  const onResize = (key: number, size: number) =>
    setRoot((r) => resizeMosaicLeaf(r, key, size));

  return {
    root,
    onDrop,
    onClose,
    onSelect,
    onResize,
  };
};
