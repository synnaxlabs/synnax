import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { MosaicNode, TabEntry, Location, MosaicTree } from "@synnaxlabs/pluto";

export type MosaicSliceState = {
  mosaics: Record<string, MosaicNode>;
};

export type MosaicSlicStoreState = {
  mosaic: MosaicSliceState;
};

const initialTree: MosaicNode = {
  key: 0,
  level: 0,
  direction: "horizontal",
  first: {
    level: 1,
    key: 1,
    tabs: [
      {
        tabKey: "1",
        title: "Tab 1",
      },
      {
        tabKey: "4",
        title: "Tab 4",
      },
    ],
  },
  last: {
    level: 1,
    key: 2
    tabs: [
      {
        tabKey: "2",
        title: "Tab 2",
      },
      {
        tabKey: "3",
        title: "Tab 3",
      },
    ],
  },
};

const initialState: MosaicSliceState = {
  mosaics: {
    main: initialTree,
  },
};


const slice = createSlice({
  name: "tabs",
  initialState,
  reducers: {
,
});

export const { insertTab, removeTab, moveTab, selectTab, resize } =
  slice.actions;

export default slice;
