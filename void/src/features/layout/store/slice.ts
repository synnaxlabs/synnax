import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { MosaicNode, TabEntry, MosaicTree, Location } from "@synnaxlabs/pluto";
import { LayoutContent, LayoutPlacement } from "../types";

export interface LayoutState {
  contents: Record<string, LayoutContent<any>>;
  placements: Record<string, LayoutPlacement>;
  mosaic: MosaicNode;
}

export type LayoutStoreState = { layout: LayoutState };

const initialState: LayoutState = {
  contents: {
    main: {
      key: "main",
      title: "Void",
      type: "main",
      props: {},
    },
  },
  placements: {
    main: {
      contentKey: "main",
      location: "window",
    },
  },
  mosaic: { key: 0, level: 0 },
};

export type SetLayoutContentAction = PayloadAction<LayoutContent<any>>;
export type DeleteLayoutContentAction = PayloadAction<string>;
export type SetPlacementAction = PayloadAction<LayoutPlacement>;
export type DeletePlacementAction = PayloadAction<string>;
export type MovePlacementAction = PayloadAction<LayoutPlacement>;

type DeleteLayoutMosaicTabAction = PayloadAction<{ tabKey: string }>;

type MoveLayoutMosaicTabAction = PayloadAction<{
  tabKey: string;
  key: number;
  loc: Location;
}>;

type ResizeLayoutMosaicTabAction = PayloadAction<{ key: number; size: number }>;

type SelectLayoutMosaicTabAction = PayloadAction<{ tabKey: string }>;

export const layoutSlice = createSlice({
  name: "layout",
  initialState,
  reducers: {
    setLayoutContent: (
      { contents },
      { payload: content }: SetLayoutContentAction
    ) => {
      contents[content.key] = content;
    },
    deleteLayoutContent: (
      { mosaic, placements, contents },
      { payload: contentKey }: DeleteLayoutContentAction
    ) => {
      delete contents[contentKey];
      [mosaic, placements] = _deleteLayoutPlacement(
        contentKey,
        mosaic,
        placements
      );
    },
    deleteLayoutPlacement: (
      { placements, mosaic },
      { payload: key }: DeletePlacementAction
    ) => {
      [mosaic, placements] = _deleteLayoutPlacement(key, mosaic, placements);
    },
    setLayoutPlacement: (
      { mosaic, placements, contents },
      { payload: placement }: SetPlacementAction
    ) => {
      const { contentKey, location } = placement;
      const existingC = contents[contentKey];
      if (!existingC)
        return console.error("No content found for key", contentKey);

      const existingP = placements[contentKey];

      // If we're mosivng from a mosaic to a window, we need to remove the tab
      // from the mosaic.
      if (existingP && existingP.location === "mosaic" && location === "window")
        mosaic = _removeMosaicTab(initialState.mosaic, contentKey);

      // If we're mosivng from a window to a mosaic, we need to add the tab
      // to the mosaic.
      if (existingP.location !== "mosaic" && location == "mosaic")
        mosaic = _insertMosaicTab(mosaic, {
          tabKey: contentKey,
          title: existingC?.title,
        });

      placements[contentKey] = { contentKey, location };
    },
    deleteLayoutMosaicTab: (
      { mosaic, placements: placement, contents: content },
      { payload: { tabKey } }: DeleteLayoutMosaicTabAction
    ) => {
      mosaic = _removeMosaicTab(mosaic, tabKey);
      delete placement[tabKey];
      delete content[tabKey];
    },
    moveLayoutMosaicTab: (
      { mosaic },
      { payload: { tabKey, key, loc } }: MoveLayoutMosaicTabAction
    ) => {
      const tree = new MosaicTree(mosaic);
      tree.move(tabKey, key, loc);
      mosaic = tree.shallowCopy();
    },
    selectLayoutMosaicTab: (
      { mosaic },
      { payload: { tabKey } }: SelectLayoutMosaicTabAction
    ) => {
      const tree = new MosaicTree(mosaic);
      tree.select(tabKey);
      mosaic = tree.shallowCopy();
    },
    resizeLayoutMosaicTab: (
      { mosaic },
      { payload: { key, size } }: ResizeLayoutMosaicTabAction
    ) => {
      const tree = new MosaicTree(mosaic);
      tree.resize(key, size);
      mosaic = tree.shallowCopy();
    },
  },
});

export const {
  setLayoutContent,
  deleteLayoutContent,
  setLayoutPlacement,
  deleteLayoutPlacement,
  deleteLayoutMosaicTab,
  moveLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
} = layoutSlice.actions;

const _removeMosaicTab = (mosaic: MosaicNode, tabKey: string) => {
  const tree = new MosaicTree(mosaic);
  tree.remove(tabKey);
  return tree.shallowCopy();
};

const _insertMosaicTab = (
  mosaic: MosaicNode,
  tab: TabEntry,
  key?: number | undefined,
  loc?: Location | undefined
) => {
  const tree = new MosaicTree(mosaic);
  tree.insert(tab, key, loc);
  return tree.shallowCopy();
};

const _deleteLayoutPlacement = (
  contentKey: string,
  mosaic: MosaicNode,
  placements: Record<string, LayoutPlacement>
): [MosaicNode, Record<string, LayoutPlacement>] => {
  const existingP = placements[contentKey];
  if (existingP) {
    delete placements[contentKey];
    if (existingP.location === "mosaic") {
      mosaic = _removeMosaicTab(mosaic, contentKey);
    }
  }
  return [mosaic, placements];
};
