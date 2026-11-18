// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type schematic } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import {
  type Control,
  control,
  type Diagram,
  Properties,
  Viewport,
} from "@synnaxlabs/pluto";
import { color, control as xcontrol, narrow, sticky, xy } from "@synnaxlabs/x";
import z from "zod";

import { Window } from "@/session/window";

const viewportZ = z.object({
  position: xy.xyZ.default({ x: 0, y: 0 }),
  zoom: z.number().default(1),
  mode: Viewport.modeZ.default("select"),
});
export interface Viewport extends z.infer<typeof viewportZ> {}

const legendStateZ = z.object({
  visible: z.boolean().default(true),
  position: sticky.xyZ.default({
    x: 24,
    y: 24,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  }),
  colors: z.record(z.string(), color.colorZ).default({}),
});
export interface LegendState extends z.infer<typeof legendStateZ> {}

export const toolbarTabZ = z.enum(["symbols", "properties", "control"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;
const toolbarStateZ = z.object({
  selectedTab: toolbarTabZ.default("symbols"),
  selectedSymbolGroup: z.string().default("general"),
  /** The last tab selected in a symbol's properties form. */
  propertiesTab: Properties.tabKeyZ.optional(),
  /** Whether a click in the symbols tab changes the selected symbols. */
  changing: z.boolean().default(false),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

const controlStateZ = z.object({
  authority: xcontrol.authorityZ.default(1),
  status: control.statusZ.default("released"),
});

const stateZ = z.object({
  control: controlStateZ.prefault({}),
  selected: z.array(z.string()).default([]),
  legend: legendStateZ.prefault({}),
  toolbar: toolbarStateZ.prefault({}),
  editable: z.boolean().default(true),
  fitViewOnResize: z.boolean().default(false),
  tooltipsDisabled: z.boolean().default(false),
  viewport: viewportZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

const windowStateZ = z.record(z.string(), stateZ).default({});

// liftV0Alpha converts a version 0 legend color alpha, stored on the 0-1 scale, to a
// byte. A non-zero alpha stays non-zero, as the Core's color migration does.
const liftV0Alpha = (a: number): number =>
  a > 0 ? Math.max(1, Math.round(Math.min(a, 1) * 255)) : 0;

// migrateV0 lifts a version 0 slice to version 1, which stores each legend color alpha
// as a 0-255 byte. Any other value passes through for the schema to judge.
const migrateV0 = (raw: unknown): unknown => {
  if (!narrow.isObject(raw) || (raw.version ?? 0) !== 0) return raw;
  if (!narrow.isObject(raw.windows)) return { ...raw, version: 1 };
  const windows = Object.fromEntries(
    Object.entries(raw.windows).map(([windowKey, docs]) => {
      if (!narrow.isObject(docs)) return [windowKey, docs];
      return [
        windowKey,
        Object.fromEntries(
          Object.entries(docs).map(([docKey, doc]) => {
            if (!narrow.isObject(doc) || !narrow.isObject(doc.legend))
              return [docKey, doc];
            const colors = doc.legend.colors;
            if (!narrow.isObject(colors)) return [docKey, doc];
            const lifted = Object.fromEntries(
              Object.entries(colors).map(([k, c]) => [
                k,
                Array.isArray(c) && c.length === 4 && typeof c[3] === "number"
                  ? [c[0], c[1], c[2], liftV0Alpha(c[3])]
                  : c,
              ]),
            );
            return [docKey, { ...doc, legend: { ...doc.legend, colors: lifted } }];
          }),
        ),
      ];
    }),
  );
  return { ...raw, version: 1, windows };
};

export const sliceStateZ = z.preprocess(
  migrateV0,
  z.object({
    version: z.literal(1).default(1),
    /** A window is a viewport, so each holds its own view of every document. */
    windows: z.record(z.string(), windowStateZ).default({}),
  }),
);

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "schematic";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload extends Window.OptionalKeyParams {
  key: schematic.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetSelectedPayload extends KeyedPayload {
  selected: string[];
}

export interface SetControlStatusPayload extends KeyedPayload {
  status: Control.Status;
}

export interface SetAuthorityPayload extends KeyedPayload {
  authority: xcontrol.Authority;
}

export interface MoveLegendPayload extends KeyedPayload {
  position: sticky.XY;
}

export interface SetLegendColorsPayload extends KeyedPayload {
  colors: Record<string, color.Color>;
}

export interface SetLegendVisiblePayload extends KeyedPayload {
  visible: boolean;
}

export interface SelectToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface SetSelectedSymbolGroupPayload extends KeyedPayload {
  group: string;
}

export interface SetPropertiesTabPayload extends KeyedPayload {
  tab: Properties.TabKey;
}

export interface SetChangingPayload extends KeyedPayload {
  changing: boolean;
}

export interface SetEditablePayload extends KeyedPayload {
  editable: boolean;
}

export interface SetFitViewOnResizePayload extends KeyedPayload {
  fitViewOnResize: boolean;
}

export interface SetTooltipsDisabledPayload extends KeyedPayload {
  tooltipsDisabled: boolean;
}

export interface SetViewportPayload extends KeyedPayload {
  viewport: Diagram.Viewport;
}

export interface SetViewportModePayload extends KeyedPayload {
  mode: Viewport.Mode;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

// Properties only configures a selection, so emptying one leaves that tab.
const clearSelection = (state: State): void => {
  state.selected = [];
  state.toolbar.changing = false;
  if (state.toolbar.selectedTab === "properties") state.toolbar.selectedTab = "symbols";
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setSelected: withSelectedState<SetSelectedPayload, SliceState>(
      (state, { payload }) => {
        state.selected = payload.selected;
        state.toolbar.changing = false;
        state.toolbar.selectedTab =
          payload.selected.length > 0 ? "properties" : "symbols";
      },
    ),
    setControlStatus: withSelectedState<SetControlStatusPayload, SliceState>(
      (state, { payload: { status: control } }) => {
        state.control.status = control;
        if (control !== "acquired") return;
        clearSelection(state);
        state.editable = false;
      },
    ),
    setControlAuthority: withSelectedState<SetAuthorityPayload, SliceState>(
      (state, { payload: { authority } }) => {
        state.control.authority = authority;
      },
    ),
    moveLegend: withSelectedState<MoveLegendPayload, SliceState>(
      (state, { payload: { position } }) => {
        state.legend.position = position;
      },
    ),
    setLegendColors: withSelectedState<SetLegendColorsPayload, SliceState>(
      (state, { payload: { colors } }) => {
        state.legend.colors = colors;
      },
    ),
    setLegendVisible: withSelectedState<SetLegendVisiblePayload, SliceState>(
      (state, { payload: { visible } }) => {
        state.legend.visible = visible;
      },
    ),
    selectToolbarTab: withSelectedState<SelectToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.selectedTab = tab;
        state.toolbar.changing = false;
      },
    ),
    setChanging: withSelectedState<SetChangingPayload, SliceState>(
      (state, { payload: { changing } }) => {
        state.toolbar.changing = changing;
        state.toolbar.selectedTab = changing ? "symbols" : "properties";
      },
    ),
    setSelectedSymbolGroup: withSelectedState<
      SetSelectedSymbolGroupPayload,
      SliceState
    >((state, { payload: { group } }) => {
      state.toolbar.selectedSymbolGroup = group;
    }),
    setPropertiesTab: withSelectedState<SetPropertiesTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.propertiesTab = tab;
      },
    ),
    setEditable: withSelectedState<SetEditablePayload, SliceState>(
      (state, { payload: { editable } }) => {
        state.editable = editable;
        if (!editable) clearSelection(state);
      },
    ),
    setFitViewOnResize: withSelectedState<SetFitViewOnResizePayload, SliceState>(
      (state, { payload: { fitViewOnResize } }) => {
        state.fitViewOnResize = fitViewOnResize;
      },
    ),
    setTooltipsDisabled: withSelectedState<SetTooltipsDisabledPayload, SliceState>(
      (state, { payload: { tooltipsDisabled } }) => {
        state.tooltipsDisabled = tooltipsDisabled;
      },
    ),
    setViewport: withSelectedState<SetViewportPayload, SliceState>(
      (state, { payload: { viewport } }) => {
        state.viewport = { ...state.viewport, ...viewport };
      },
    ),
    setViewportMode: withSelectedState<SetViewportModePayload, SliceState>(
      (state, { payload: { mode } }) => {
        state.viewport.mode = mode;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const {
  create,
  setSelected,
  setControlStatus,
  setControlAuthority,
  setLegendColors,
  moveLegend,
  setLegendVisible,
  selectToolbarTab,
  setSelectedSymbolGroup,
  setPropertiesTab,
  setChanging,
  setEditable,
  setFitViewOnResize,
  setTooltipsDisabled,
  setViewport,
  setViewportMode,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const purgeState = (state: State): State => {
  state.control.status = "released";
  clearSelection(state);
  return state;
};

const purgeSliceState = <S extends StoreState>(state: S): S => {
  Window.purgeDocuments(state[SLICE_NAME], purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    create,
    setSelected,
    setControlStatus,
    setControlAuthority,
    setLegendColors,
    moveLegend,
    setLegendVisible,
    selectToolbarTab,
    setSelectedSymbolGroup,
    setPropertiesTab,
    setChanging,
    setEditable,
    setFitViewOnResize,
    setTooltipsDisabled,
    setViewport,
    setViewportMode,
  ]),
];
