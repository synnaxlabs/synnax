// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type lineplot } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import { Viewport } from "@synnaxlabs/pluto";
import { lineplot as etherLineplot } from "@synnaxlabs/pluto/ether";
import { array, box, deep, dimensions, xy } from "@synnaxlabs/x";
import z from "zod";

import { Window } from "@/session/window";

export const viewportZ = z.object({
  renderTrigger: z.number().default(0),
  zoom: dimensions.dimensionsZ.default(dimensions.DECIMAL),
  pan: xy.xyZ.default(xy.ZERO),
});
export interface ViewportState extends z.infer<typeof viewportZ> {}

export const selectionZ = z.object({ box: box.box.default(box.ZERO) });
export interface SelectionState extends z.infer<typeof selectionZ> {}

export const toolbarTabZ = z.enum([
  "data",
  "lines",
  "axes",
  "properties",
  "annotations",
]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;
export const toolbarZ = z.object({ activeTab: toolbarTabZ.default("data") });
export interface ToolbarState extends z.infer<typeof toolbarZ> {}

export const clickModeZ = z.enum(["annotate", "measure"]);
export type ClickMode = z.infer<typeof clickModeZ>;
export const controlZ = z.object({
  hold: z.boolean().default(false),
  clickMode: clickModeZ.nullable().default(null),
  enableTooltip: z.boolean().default(true),
});
export interface ControlState extends z.infer<typeof controlZ> {}

export const measureZ = z.object({
  mode: etherLineplot.measure.modeZ.default("one"),
});
export interface MeasureState extends z.infer<typeof measureZ> {}

export const annotationsZ = z.object({ visible: z.boolean().default(true) });
export interface AnnotationsState extends z.infer<typeof annotationsZ> {}

export const stateZ = z.object({
  viewport: viewportZ.prefault({}),
  selection: selectionZ.prefault({}),
  mode: Viewport.modeZ.default("zoom"),
  control: controlZ.prefault({}),
  toolbar: toolbarZ.prefault({}),
  measure: measureZ.prefault({}),
  annotations: annotationsZ.prefault({}),
  selectedRules: z.array(z.string()).default([]),
  hiddenLines: z.array(z.string()).default([]),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});
export const ZERO_ANNOTATIONS_STATE = annotationsZ.parse({});

export const windowStateZ = z.record(z.string(), stateZ).default({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** A window is a viewport, so each holds its own view of every document. */
  windows: z.record(z.string(), windowStateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "line_plot";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload extends Window.OptionalKeyParams {
  key: lineplot.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetViewportPayload
  extends KeyedPayload, Partial<Omit<ViewportState, "renderTrigger">> {}

export interface StoreViewportPayload
  extends KeyedPayload, Omit<ViewportState, "renderTrigger"> {}

export interface SetSelectionPayload extends KeyedPayload, SelectionState {}

export interface SetActiveToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface SetControlHoldPayload extends KeyedPayload {
  hold?: boolean;
}

export interface ToggleControlClickModePayload extends KeyedPayload {
  mode: ClickMode;
}

export interface SetControlEnableTooltipPayload extends KeyedPayload {
  enabled?: boolean;
}

export interface SetViewportModePayload extends KeyedPayload {
  mode: Viewport.Mode;
}

export interface SetSelectedRulePayload extends KeyedPayload {
  ruleKey: string | string[];
}

export interface SetMeasureModePayload extends KeyedPayload {
  mode: etherLineplot.measure.Mode;
}

export interface SetRangeAnnotationsVisiblePayload extends KeyedPayload {
  visible: boolean;
}

export interface SetLineVisiblePayload extends KeyedPayload {
  lineKey: string;
  visible: boolean;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setViewport: withSelectedState<SetViewportPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, ...rest } = payload;
        state.viewport = {
          ...deep.copy(ZERO_STATE.viewport),
          ...rest,
          renderTrigger: state.viewport.renderTrigger + 1,
        };
        state.selection = deep.copy(ZERO_STATE.selection);
      },
    ),
    storeViewport: withSelectedState<StoreViewportPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, ...rest } = payload;
        state.viewport = { ...state.viewport, ...rest };
        state.selection = deep.copy(ZERO_STATE.selection);
      },
    ),
    setSelection: withSelectedState<SetSelectionPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, ...rest } = payload;
        state.selection = { ...state.selection, ...rest };
      },
    ),
    setActiveToolbarTab: withSelectedState<SetActiveToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.activeTab = tab;
      },
    ),
    setControlHold: withSelectedState<SetControlHoldPayload, SliceState>(
      (state, { payload: { hold } }) => {
        state.control.hold = hold !== undefined ? hold : !state.control.hold;
      },
    ),
    toggleControlClickMode: withSelectedState<
      ToggleControlClickModePayload,
      SliceState
    >((state, { payload: { mode } }) => {
      state.control.clickMode = state.control.clickMode === mode ? null : mode;
    }),
    setControlEnableTooltip: withSelectedState<
      SetControlEnableTooltipPayload,
      SliceState
    >((state, { payload: { enabled } }) => {
      state.control.enableTooltip =
        enabled !== undefined ? enabled : !state.control.enableTooltip;
    }),
    setViewportMode: withSelectedState<SetViewportModePayload, SliceState>(
      (state, { payload: { mode } }) => {
        state.mode = mode;
      },
    ),
    setSelectedRule: withSelectedState<SetSelectedRulePayload, SliceState>(
      (state, { payload: { ruleKey } }) => {
        state.selectedRules = array.toArray(ruleKey);
        state.toolbar.activeTab = "annotations";
      },
    ),
    setMeasureMode: withSelectedState<SetMeasureModePayload, SliceState>(
      (state, { payload: { mode } }) => {
        state.measure.mode = mode;
      },
    ),
    setRangeAnnotationsVisible: withSelectedState<
      SetRangeAnnotationsVisiblePayload,
      SliceState
    >((state, { payload: { visible } }) => {
      state.annotations.visible = visible;
    }),
    setLineVisible: withSelectedState<SetLineVisiblePayload, SliceState>(
      (state, { payload: { lineKey, visible } }) => {
        const hidden = new Set(state.hiddenLines);
        if (visible) hidden.delete(lineKey);
        else hidden.add(lineKey);
        state.hiddenLines = Array.from(hidden);
      },
    ),
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const {
  create,
  setViewport,
  storeViewport,
  setSelection,
  setActiveToolbarTab,
  setControlHold,
  toggleControlClickMode,
  setControlEnableTooltip,
  setViewportMode,
  setSelectedRule,
  setMeasureMode,
  setRangeAnnotationsVisible,
  setLineVisible,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const purgeState = (state: State): State => {
  state.selectedRules = [];
  return state;
};

export const purgeSliceState = <S extends StoreState>(state: S): S => {
  Window.purgeDocuments(state[SLICE_NAME], purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    create,
    setViewport,
    storeViewport,
    setSelection,
    setActiveToolbarTab,
    setControlHold,
    toggleControlClickMode,
    setControlEnableTooltip,
    setViewportMode,
    setSelectedRule,
    setMeasureMode,
    setRangeAnnotationsVisible,
    setLineVisible,
  ]),
];
