// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { schematic } from "@synnaxlabs/client";
import {
  Access,
  type Control,
  control,
  type Diagram,
  Schematic,
  Viewport,
} from "@synnaxlabs/pluto";
import { color, control as xcontrol, sticky, xy } from "@synnaxlabs/x";
import z from "zod";

import { Keyed } from "@/keyed";
import { type RootState } from "@/store";

export const viewportZ = z.object({
  position: xy.xyZ.default({ x: 0, y: 0 }),
  zoom: z.number().default(1),
  mode: Viewport.modeZ.default("select"),
});
export interface Viewport extends z.infer<typeof viewportZ> {}

export const legendStateZ = z.object({
  visible: z.boolean().default(true),
  position: sticky.xyZ.default({ x: 100, y: 100 }),
  colors: z.record(z.string(), color.colorZ).default({}),
});
export interface LegendState extends z.infer<typeof legendStateZ> {}

export const toolbarTabZ = z.enum(["symbols", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;
export const toolbarStateZ = z.object({
  selectedTab: toolbarTabZ.default("symbols"),
  selectedSymbolGroup: z.string().default("general"),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const stateZ = z.object({
  selected: z.array(z.string()).default([]),
  controlStatus: control.statusZ.default("released"),
  authority: xcontrol.authorityZ.default(100),
  legend: legendStateZ.prefault({}),
  toolbar: toolbarStateZ.prefault({}),
  editable: z.boolean().default(false),
  fitViewOnResize: z.boolean().default(false),
  viewport: viewportZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const sliceStateZ = z.object({
  schematics: z.record(z.string(), stateZ),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload extends Keyed.Payload, NewState {}

export interface SetSelectedPayload extends Keyed.Payload {
  selected: string[];
}

export interface SetControlStatusPayload extends Keyed.Payload {
  control: Control.Status;
}

export interface SetAuthorityPayload extends Keyed.Payload {
  authority: xcontrol.Authority;
}

export interface MoveLegendPayload extends Keyed.Payload {
  position: xy.XY;
}

export interface SetLegendColorsPayload extends Keyed.Payload {
  colors: Record<string, color.Color>;
}

export interface SetLegendVisiblePayload extends Keyed.Payload {
  visible: boolean;
}

export interface SelectToolbarTabPayload extends Keyed.Payload {
  tab: ToolbarTab;
}

export interface SetSelectedSymbolGroupPayload extends Keyed.Payload {
  group: string;
}

export interface SetEditablePayload extends Keyed.Payload {
  editable: boolean;
}

export interface SetFitViewOnResizePayload extends Keyed.Payload {
  fitViewOnResize: boolean;
}

export interface SetViewportPayload extends Keyed.Payload {
  viewport: Diagram.Viewport;
}

export interface SetViewportModePayload extends Keyed.Payload {
  mode: Viewport.Mode;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState =
  <Payload extends Keyed.Payload, Type extends string = string>(
    handler: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.schematics[key];
    if (s == null) {
      s = stateZ.parse({});
      state.schematics[key] = s;
    }
    handler(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    create: (state, { payload: { key, ...rest } }: PayloadAction<CreatePayload>) => {
      if (!(key in state.schematics)) state.schematics[key] = stateZ.parse(rest);
    },
    setSelected: withSelectedState(
      (s, { payload: { selected } }: PayloadAction<SetSelectedPayload>) => {
        s.selected = selected;
        s.toolbar.selectedTab = selected.length > 0 ? "properties" : "symbols";
      },
    ),
    setControlStatus: withSelectedState(
      (state, { payload: { control } }: PayloadAction<SetControlStatusPayload>) => {
        state.controlStatus = control;
        if (control !== "acquired") return;
        state.selected = [];
        state.editable = false;
      },
    ),
    setAuthority: withSelectedState(
      (state, { payload: { authority } }: PayloadAction<SetAuthorityPayload>) => {
        state.authority = authority;
      },
    ),
    moveLegend: withSelectedState(
      (state, { payload: { position } }: PayloadAction<MoveLegendPayload>) => {
        state.legend.position = position;
      },
    ),
    setLegendColors: withSelectedState(
      (state, { payload: { colors } }: PayloadAction<SetLegendColorsPayload>) => {
        state.legend.colors = colors;
      },
    ),
    setLegendVisible: withSelectedState(
      (state, { payload: { visible } }: PayloadAction<SetLegendVisiblePayload>) => {
        state.legend.visible = visible;
      },
    ),
    selectToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SelectToolbarTabPayload>) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setSelectedSymbolGroup: withSelectedState(
      (state, { payload: { group } }: PayloadAction<SetSelectedSymbolGroupPayload>) => {
        state.toolbar.selectedSymbolGroup = group;
      },
    ),
    setEditable: withSelectedState(
      (state, { payload: { editable } }: PayloadAction<SetEditablePayload>) => {
        state.editable = editable;
        if (!editable) state.selected = [];
      },
    ),
    setFitViewOnResize: withSelectedState(
      (
        state,
        { payload: { fitViewOnResize } }: PayloadAction<SetFitViewOnResizePayload>,
      ) => {
        state.fitViewOnResize = fitViewOnResize;
      },
    ),
    setViewport: withSelectedState(
      (state, { payload }: PayloadAction<SetViewportPayload>) => {
        state.viewport = { ...state.viewport, ...payload.viewport };
      },
    ),
    setViewportMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetViewportModePayload>) => {
        state.viewport.mode = mode;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.schematics[key]);
    },
  },
});

export const {
  create: internalCreate,
  setLegendColors,
  setSelected,
  setControlStatus,
  setAuthority,
  moveLegend,
  setLegendVisible,
  selectToolbarTab,
  setSelectedSymbolGroup,
  setEditable,
  setFitViewOnResize,
  setViewport,
  setViewportMode,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const purgeState = (state: State): State => {
  state.controlStatus = "released";
  state.toolbar = { ...state.toolbar, selectedTab: "symbols" };
  state.selected = [];
  return state;
};

export const purgeSliceState = (state: RootState): RootState => {
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectSelected = (state: StoreState, key: string): string[] =>
  select(state, key).selected;

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).schematics[key] ?? stateZ.parse({});

export const selectControlStatus = (state: StoreState, key: string): Control.Status =>
  select(state, key)?.controlStatus ?? "released";

export const selectAuthority = (state: StoreState, key: string): xcontrol.Authority =>
  select(state, key)?.authority ?? 1;

export const selectToolbar = (state: StoreState, key: string): ToolbarState =>
  select(state, key).toolbar;

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  selectToolbar(state, key).selectedTab;

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectToolbar(state, key).selectedSymbolGroup;

export const selectLegend = (state: StoreState, key: string): LegendState =>
  select(state, key)?.legend;

export const selectLegendVisible = (state: StoreState, key: string): boolean =>
  select(state, key).legend.visible;

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const selectFitViewOnResize = (state: StoreState, key: string): boolean =>
  select(state, key).fitViewOnResize;

export const selectViewport = (state: StoreState, key: string): Viewport =>
  select(state, key).viewport;

const { useSelectEditableBase, ...selectors } = Keyed.createSelectors(
  Schematic.useKey,
  {
    useSelectState: select,
    useSelectSelected: selectSelected,
    useSelectControlStatus: selectControlStatus,
    useSelectAuthority: selectAuthority,
    useSelectActiveToolbarTab: selectActiveToolbarTab,
    useSelectToolbar: selectToolbar,
    useSelectSelectedSymbolGroup: selectSelectedSymbolGroup,
    useSelectLegend: selectLegend,
    useSelectLegendVisible: selectLegendVisible,
    useSelectEditableBase: selectEditable,
    useSelectFitViewOnResize: selectFitViewOnResize,
    useSelectViewport: selectViewport,
  },
);

export const useSelectEditable = (): [boolean, boolean] => {
  const key = Schematic.useKey();
  const isSnapshot = Schematic.useSelectSnapshot({});
  const hasUpdatePermission = Access.useUpdateGranted(schematic.ontologyID(key));
  const editable = useSelectEditableBase();
  const canEdit = hasUpdatePermission && !isSnapshot;
  return [canEdit && editable, canEdit];
};

export const {
  useSelectState,
  useSelectSelected,
  useSelectControlStatus,
  useSelectAuthority,
  useSelectActiveToolbarTab,
  useSelectToolbar,
  useSelectSelectedSymbolGroup,
  useSelectLegend,
  useSelectLegendVisible,
  useSelectFitViewOnResize,
  useSelectViewport,
} = selectors;

export const {
  useSetSelected,
  useSetViewport,
  useSetEditable,
  useSetFitViewOnResize,
  useSetViewportMode,
  useSetLegendColors,
  useMoveLegend,
  useSetControlStatus,
  useSetControlAuthority,
  useSetLegendVisible,
  useSelectToolbarTab,
} = Keyed.createDispatchHandlers(Schematic.useKey, {
  useSetSelected: ["selected", setSelected],
  useSetViewport: ["viewport", setViewport],
  useSetEditable: ["editable", setEditable],
  useSetFitViewOnResize: ["fitViewOnResize", setFitViewOnResize],
  useSetViewportMode: ["mode", setViewportMode],
  useSetLegendColors: ["colors", setLegendColors],
  useMoveLegend: ["position", moveLegend],
  useSetControlStatus: ["control", setControlStatus],
  useSetControlAuthority: ["authority", setAuthority],
  useSetLegendVisible: ["visible", setLegendVisible],
  useSelectToolbarTab: ["tab", selectToolbarTab],
});
