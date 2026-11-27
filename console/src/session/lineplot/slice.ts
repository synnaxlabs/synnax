// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { lineplot } from "@synnaxlabs/client";
import type { Drift } from "@synnaxlabs/drift";
import { Viewport } from "@synnaxlabs/pluto";
import { lineplot as etherLineplot } from "@synnaxlabs/pluto/ether";
import { array, box, deep, dimensions, xy } from "@synnaxlabs/x";
import z from "zod";

import { Window } from "@/session/window";

const viewportZ = z.object({
  renderTrigger: z.number().default(0),
  zoom: dimensions.dimensionsZ.default(dimensions.DECIMAL),
  pan: xy.xyZ.default(xy.ZERO),
});
export interface ViewportState extends z.infer<typeof viewportZ> {}

const selectionZ = z.object({ box: box.box.default(box.ZERO) });
export interface SelectionState extends z.infer<typeof selectionZ> {}

export const toolbarTabZ = z.enum([
  "data",
  "lines",
  "axes",
  "properties",
  "annotations",
]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;
const toolbarZ = z.object({ activeTab: toolbarTabZ.default("data") });
export interface ToolbarState extends z.infer<typeof toolbarZ> {}

export const clickModeZ = z.enum(["annotate", "measure"]);
export type ClickMode = z.infer<typeof clickModeZ>;
const controlZ = z.object({
  hold: z.boolean().default(false),
  clickMode: clickModeZ.nullable().default(null),
  enableTooltip: z.boolean().default(true),
});
export interface ControlState extends z.infer<typeof controlZ> {}

const measureZ = z.object({
  mode: etherLineplot.measure.modeZ.default("one"),
});

const annotationsZ = z.object({ visible: z.boolean().default(true) });

// The open breaks are the lower bounds of their gaps. The viewport and the selection
// hold the times of the layout with them open.
const breaksZ = z.object({
  hidden: z.boolean().default(false),
  open: z.array(z.number()).default([]),
});

const hiddenZ = z.object({
  lines: z.array(z.string()),
  ranges: z.array(z.string()),
  traces: z.array(z.string()),
});
/** The items of a plot that do not show. */
export interface Hidden extends z.infer<typeof hiddenZ> {}

const isolationZ = z.object({ item: z.string(), restore: hiddenZ });
/** The item the user isolated, and what was hidden before. */
export interface Isolation extends z.infer<typeof isolationZ> {}

const stateZ = z.object({
  viewport: viewportZ.prefault({}),
  selection: selectionZ.prefault({}),
  mode: Viewport.modeZ.default("zoom"),
  control: controlZ.prefault({}),
  toolbar: toolbarZ.prefault({}),
  measure: measureZ.prefault({}),
  annotations: annotationsZ.prefault({}),
  breaks: breaksZ.prefault({}),
  selectedRules: z.array(z.string()).default([]),
  hiddenLines: z.array(z.string()).default([]),
  /** The keys of the ranges that do not show on a relative x axis. */
  hiddenRanges: z.array(z.string()).default([]),
  /** The keys of the lines of one range that do not show on a relative x axis. */
  hiddenTraces: z.array(z.string()).default([]),
  /** The item the user isolated, if any. */
  isolation: isolationZ.nullable().default(null),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

const windowStateZ = z.record(z.string(), stateZ).default({});

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

export interface StoreBreaksPayload extends StoreViewportPayload {
  open: number[];
  selection: box.Box;
}

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

export interface SetBreaksHiddenPayload extends KeyedPayload {
  hidden: boolean;
}

export interface SetHiddenPayload extends KeyedPayload {
  hidden: Hidden;
  isolation: Isolation | null;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

// resetView moves the plot to its home view with every break closed. It bumps the
// render trigger, so a mounted plot applies the view.
const resetView = (state: State, viewport: Partial<ViewportState> = {}): void => {
  state.viewport = {
    ...deep.copy(ZERO_STATE.viewport),
    ...viewport,
    renderTrigger: state.viewport.renderTrigger + 1,
  };
  state.selection = deep.copy(ZERO_STATE.selection);
  state.breaks.open = [];
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setViewport: withSelectedState<SetViewportPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, windowKey: _windowKey, ...rest } = payload;
        resetView(state, rest);
      },
    ),
    storeViewport: withSelectedState<StoreViewportPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, windowKey: _windowKey, ...rest } = payload;
        state.viewport = { ...state.viewport, ...rest };
        state.selection = deep.copy(ZERO_STATE.selection);
      },
    ),
    setSelection: withSelectedState<SetSelectionPayload, SliceState>(
      (state, { payload }) => {
        const { key: _key, windowKey: _windowKey, ...rest } = payload;
        state.selection = { ...state.selection, ...rest };
      },
    ),
    storeBreaks: withSelectedState<StoreBreaksPayload, SliceState>(
      (state, { payload: { open, pan, zoom, selection } }) => {
        state.breaks.open = open;
        state.viewport = { ...state.viewport, pan, zoom };
        state.selection = { box: selection };
      },
    ),
    setActiveToolbarTab: withSelectedState<SetActiveToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.activeTab = tab;
      },
    ),
    setControlHold: withSelectedState<SetControlHoldPayload, SliceState>(
      (state, { payload: { hold } }) => {
        state.control.hold = hold ?? !state.control.hold;
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
      state.control.enableTooltip = enabled ?? !state.control.enableTooltip;
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
    setBreaksHidden: withSelectedState<SetBreaksHiddenPayload, SliceState>(
      (state, { payload: { hidden } }) => {
        state.breaks.hidden = hidden;
        // The breaks change the home view, so the zoom resets to show the whole plot.
        resetView(state);
      },
    ),
    setHidden: withSelectedState<SetHiddenPayload, SliceState>(
      (state, { payload: { hidden, isolation } }) => {
        state.hiddenLines = hidden.lines;
        state.hiddenRanges = hidden.ranges;
        state.hiddenTraces = hidden.traces;
        state.isolation = isolation;
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
  storeBreaks,
  setActiveToolbarTab,
  setControlHold,
  toggleControlClickMode,
  setControlEnableTooltip,
  setViewportMode,
  setSelectedRule,
  setMeasureMode,
  setRangeAnnotationsVisible,
  setBreaksHidden,
  setHidden,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const purgeState = (state: State): State => {
  state.selectedRules = [];
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
    setViewport,
    storeViewport,
    setSelection,
    storeBreaks,
    setActiveToolbarTab,
    setControlHold,
    toggleControlClickMode,
    setControlEnableTooltip,
    setViewportMode,
    setSelectedRule,
    setMeasureMode,
    setRangeAnnotationsVisible,
    setBreaksHidden,
    setHidden,
  ]),
];
