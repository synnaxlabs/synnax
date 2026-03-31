// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Control } from "@synnaxlabs/pluto";
import { type record, type sticky, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const SLICE_NAME = "schematic";

export const toolbarTabZ = z.enum(["symbols", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export interface LegendState {
  visible: boolean;
  position: sticky.XY;
}

export interface CopyBuffer {
  pos: xy.XY;
  nodes: Array<{ key: string; position: xy.XY }>;
  edges: Array<{
    key: string;
    source: { node: string; param: string };
    target: { node: string; param: string };
  }>;
  props: Record<string, record.Unknown>;
}

const ZERO_COPY_BUFFER: CopyBuffer = {
  pos: xy.ZERO,
  nodes: [],
  edges: [],
  props: {},
};

export interface State {
  selected: string[];
  control: Control.Status;
  legend: LegendState;
  activeToolbarTab: ToolbarTab;
  selectedSymbolGroup: string;
}

export const ZERO_STATE: State = {
  selected: [],
  control: "released",
  legend: { visible: false, position: { x: 0, y: 0 } },
  activeToolbarTab: "symbols",
  selectedSymbolGroup: "general",
};

export interface SliceState {
  schematics: Record<string, State>;
  copy: CopyBuffer;
}

export const ZERO_SLICE_STATE: SliceState = {
  schematics: {},
  copy: ZERO_COPY_BUFFER,
};

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const ensure = (state: SliceState, key: string): State => {
  state.schematics[key] ??= { ...ZERO_STATE };
  return state.schematics[key];
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setSelected: (
      state,
      {
        payload: { key, selected },
      }: PayloadAction<{ key: string; selected: string[] }>,
    ) => {
      const s = ensure(state, key);
      s.selected = selected;
      s.activeToolbarTab = selected.length > 0 ? "properties" : "symbols";
    },
    setControlStatus: (
      state,
      {
        payload: { key, control },
      }: PayloadAction<{ key: string; control: Control.Status }>,
    ) => {
      ensure(state, key).control = control;
    },
    setLegend: (
      state,
      {
        payload: { key, legend },
      }: PayloadAction<{ key: string; legend: Partial<LegendState> }>,
    ) => {
      const s = ensure(state, key);
      s.legend = { ...s.legend, ...legend };
    },
    setLegendVisible: (
      state,
      { payload: { key, visible } }: PayloadAction<{ key: string; visible: boolean }>,
    ) => {
      ensure(state, key).legend.visible = visible;
    },
    setActiveToolbarTab: (
      state,
      { payload: { key, tab } }: PayloadAction<{ key: string; tab: ToolbarTab }>,
    ) => {
      ensure(state, key).activeToolbarTab = tab;
    },
    setSelectedSymbolGroup: (
      state,
      { payload: { key, group } }: PayloadAction<{ key: string; group: string }>,
    ) => {
      ensure(state, key).selectedSymbolGroup = group;
    },
    copySelection: (
      state,
      {
        payload: { key, nodes, edges, props },
      }: PayloadAction<{
        key: string;
        nodes: CopyBuffer["nodes"];
        edges: CopyBuffer["edges"];
        props: CopyBuffer["props"];
      }>,
    ) => {
      const selected = new Set(ensure(state, key).selected);
      const selectedNodes = nodes.filter((n) => selected.has(n.key));
      const selectedEdges = edges.filter((e) => selected.has(e.key));
      const selectedProps: Record<string, record.Unknown> = {};
      for (const n of selectedNodes)
        if (props[n.key] != null) selectedProps[n.key] = props[n.key];
      for (const e of selectedEdges)
        if (props[e.key] != null) selectedProps[e.key] = props[e.key];
      let pos = xy.ZERO;
      if (selectedNodes.length > 0) {
        pos = selectedNodes.reduce((acc, n) => xy.translate(acc, n.position), xy.ZERO);
        pos = xy.scale(pos, 1 / selectedNodes.length);
      }
      state.copy = {
        pos,
        nodes: selectedNodes,
        edges: selectedEdges,
        props: selectedProps,
      };
    },
    remove: (state, { payload: { keys } }: PayloadAction<{ keys: string[] }>) => {
      keys.forEach((key) => delete state.schematics[key]);
    },
  },
});

export const {
  setSelected,
  setControlStatus,
  setLegend,
  setLegendVisible,
  setActiveToolbarTab,
  setSelectedSymbolGroup,
  copySelection,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
