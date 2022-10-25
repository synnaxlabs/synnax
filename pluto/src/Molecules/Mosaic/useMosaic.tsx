import { useState } from "react";
import { TabEntry } from "../../Atoms/Tabs";
import MosaicTree, { MosaicNode } from "./MosaicTree";
import { Location } from "../../util/spatial";

export interface UseMosaicProps {
  initialTree: MosaicNode;
}

export interface UseMosaicReturn {
  tree: MosaicNode;
  onDrop: (key: number, tabKey: string, loc: Location) => void;
  onClose: (tabKey: string) => void;
  insertTab: (tab: TabEntry, key?: number, loc?: Location) => void;
  onSelect: (tabKey: string) => void;
  onResize: (key: number, sizes: number) => void;
}

export const useMosaic = ({ initialTree }: UseMosaicProps): UseMosaicReturn => {
  const [tree, setTree] = useState(initialTree);

  const onDrop = (key: number, tabKey: string, loc: Location) => {
    const t = new MosaicTree(tree);
    t.move(tabKey, key, loc);
    setTree(t.shallowCopy());
  };

  const onClose = (tabKey: string) => {
    const t = new MosaicTree(tree);
    t.remove(tabKey);
    setTree(t.shallowCopy());
  };

  const insertTab = (tab: TabEntry, key?: number, loc?: Location) => {
    if (!loc) loc = "center";
    const t = new MosaicTree(tree);
    t.insert(tab, key, loc);
    setTree(t.shallowCopy());
  };

  const onSelect = (tabKey: string) => {
    const t = new MosaicTree(tree);
    t.select(tabKey);
    setTree(t.shallowCopy());
  };

  const onResize = (key: number, size: number) => {
    const t = new MosaicTree(tree);
    t.resize(key, size);
    setTree(t.shallowCopy());
  };

  return {
    tree,
    onDrop,
    onClose,
    insertTab,
    onSelect,
    onResize,
  };
};

export default useMosaic;
