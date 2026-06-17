// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { LinePlot as Base, Viewport } from "@synnaxlabs/pluto";
import { lineplot } from "@synnaxlabs/pluto/ether";
import { box, dimensions, unique, xy } from "@synnaxlabs/x";
import z from "zod";

import { Keyed } from "@/keyed";

export const clickModeZ = z.enum(["annotate", "measure"]).optional();
export type ClickMode = z.infer<typeof clickModeZ>;

export const toolbarTabZ = z.enum([
  "data",
  "lines",
  "axes",
  "annotations",
  "properties",
]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

const stateZ = z.object({
  selectedRules: z.string().array().default([]),
  hiddenLines: z.string().array().default([]),
  showRangeAnnotations: z.boolean().default(false),
  measureMode: lineplot.measure.modeZ.default("one"),
  viewportMode: Viewport.modeZ.default("zoom"),
  selectedToolbarTab: toolbarTabZ.default("data"),
  zoom: dimensions.dimensionsZ.default({ width: 1, height: 1 }),
  pan: xy.xyZ.default({ x: 0, y: 0 }),
  selection: box.box.optional(),
  hold: z.boolean().default(false),
  clickMode: clickModeZ.optional(),
  enableTooltip: z.boolean().default(true),
});

export interface State extends z.output<typeof stateZ> {}
export interface NewState extends z.output<typeof stateZ> {}

export const sliceStateZ = z.object({
  plots: z.record(z.string(), stateZ).prefault({}),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export const SLICE_NAME = "line";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload extends Keyed.Payload, NewState {}

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload extends Keyed.Payload {
  zoom: dimensions.Dimensions;
  pan: xy.XY;
}

export interface ResetViewportPayload extends Keyed.Payload {}

export interface SetSelectionPayload extends Keyed.Payload {
  selection: box.Box;
}

export interface SetActiveToolbarTabPayload extends Keyed.Payload {
  tab: ToolbarTab;
}

export interface SetClickModePayload extends Keyed.Payload {
  mode: ClickMode;
}

export interface SetHoldPayload extends Keyed.Payload {
  hold: boolean;
}

export interface SetTooltipsVisiblePayload extends Keyed.Payload {
  visible: boolean;
}

export interface SetViewportModePayload extends Keyed.Payload {
  mode: Viewport.Mode;
}

export interface SelectRulePayload extends Keyed.Payload {
  ruleKey: string;
}

export interface SetMeasureModePayload extends Keyed.Payload {
  mode: lineplot.measure.Mode;
}

export interface SetRangeAnnotationsVisiblePayload extends Keyed.Payload {
  visible: boolean;
}

export interface SetLineVisiblePayload extends Keyed.Payload {
  lineKey: string;
  visible: boolean;
}

export interface SetSelectedRulesPayload extends Keyed.Payload {
  ruleKeys: string[];
}

export interface SelectRulePayload extends Keyed.Payload {
  ruleKey: string;
}

const withSelectedState =
  <Payload extends Keyed.Payload, Type extends string = string>(
    handler: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.plots[key];
    if (s == null) {
      s = stateZ.parse({});
      state.plots[key] = s;
    }
    handler(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    create: (state, { payload: { key, ...rest } }: PayloadAction<CreatePayload>) => {
      if (!(key in state.plots)) state.plots[key] = stateZ.parse(rest);
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((k) => delete state.plots[k]);
    },
    setViewport: withSelectedState(
      (state, { payload: { zoom, pan } }: PayloadAction<SetViewportPayload>) => {
        state.zoom = zoom;
        state.pan = pan;
      },
    ),
    resetViewport: withSelectedState((state) => {
      state.zoom = { width: 1, height: 1 };
      state.pan = { x: 0, y: 0 };
    }),
    setSelection: withSelectedState(
      (state, { payload: { selection } }: PayloadAction<SetSelectionPayload>) => {
        state.selection = selection;
      },
    ),
    setSelectedToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SetActiveToolbarTabPayload>) => {
        state.selectedToolbarTab = tab;
      },
    ),
    setClickMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetClickModePayload>) => {
        state.clickMode = mode;
      },
    ),
    setViewportMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetViewportModePayload>) => {
        state.viewportMode = mode;
      },
    ),
    setHold: withSelectedState(
      (state, { payload: { hold } }: PayloadAction<SetHoldPayload>) => {
        state.hold = hold;
      },
    ),
    setSelectedRules: withSelectedState(
      (state, { payload: { ruleKeys } }: PayloadAction<SetSelectedRulesPayload>) => {
        state.selectedRules = ruleKeys;
        state.selectedToolbarTab = "annotations";
      },
    ),
    selectRule: withSelectedState(
      (state, { payload: { ruleKey } }: PayloadAction<SelectRulePayload>) => {
        state.selectedRules = unique.unique([...state.selectedRules, ruleKey]);
        state.selectedToolbarTab = "annotations";
      },
    ),
    setMeasureMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetMeasureModePayload>) => {
        state.measureMode = mode;
      },
    ),
    setRangeAnnotationsVisible: withSelectedState(
      (
        state,
        { payload: { visible } }: PayloadAction<SetRangeAnnotationsVisiblePayload>,
      ) => {
        state.showRangeAnnotations = visible;
      },
    ),
    setTooltipsVisible: withSelectedState(
      (state, { payload: { visible } }: PayloadAction<SetTooltipsVisiblePayload>) => {
        state.showRangeAnnotations = visible;
      },
    ),
    setLineVisible: withSelectedState(
      (
        state,
        { payload: { visible, lineKey } }: PayloadAction<SetLineVisiblePayload>,
      ) => {
        const hidden = new Set(state.hiddenLines);
        if (visible) hidden.delete(lineKey);
        else hidden.add(lineKey);
        state.hiddenLines = Array.from(hidden);
      },
    ),
  },
});

export const {
  create,
  remove,
  setViewport,
  setSelectedToolbarTab,
  setClickMode,
  setViewportMode,
  setSelectedRules,
  setMeasureMode,
  setRangeAnnotationsVisible,
  setLineVisible,
  selectRule,
  setHold,
  setSelection,
  setTooltipsVisible,
  resetViewport,
  setRangeAnnotationsVisible,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).plots[key] ?? stateZ.parse({});

export const selectSelectedToolbarTab = (state: StoreState, key: string): string =>
  select(state, key).selectedToolbarTab;

export const selectClickMode = (state: StoreState, key: string): ClickMode =>
  select(state, key).clickMode;

export const selectViewportMode = (state: StoreState, key: string): Viewport.Mode =>
  select(state, key).viewportMode;

export const selectViewportZoom = (
  state: StoreState,
  key: string,
): dimensions.Dimensions => select(state, key).zoom;

export const selectViewportPan = (state: StoreState, key: string): xy.XY =>
  select(state, key).pan;

export const selectHold = (state: StoreState, key: string): boolean =>
  select(state, key).hold;

export const selectEnableTooltip = (state: StoreState, key: string): boolean =>
  select(state, key).enableTooltip;

export const selectHiddenLines = (state: StoreState, key: string): string[] =>
  select(state, key).hiddenLines;

export const selectMeasureMode = (
  state: StoreState,
  key: string,
): lineplot.measure.Mode => select(state, key).measureMode;

export const selectViewportSelection = (
  state: StoreState,
  key: string,
): box.Box | undefined => select(state, key).selection;

export const selectSelectedRules = (state: StoreState, key: string): string[] =>
  select(state, key).selectedRules;

export const selectShowRangeAnnotations = (state: StoreState, key: string): boolean =>
  select(state, key).showRangeAnnotations;

export const {
  useSelect,
  useSelectSelectedToolbarTab,
  useSelectClickMode,
  useSelectViewportMode,
  useSelectHiddenLines,
  useSelectViewportSelection,
  useSelectHold,
  useSelectEnableTooltip,
  useSelectMeasureMode,
  useSelectShowRangeAnnotations,
  useSelectSelectedRules,
} = Keyed.createSelectors(Base.useKey, {
  useSelect: select,
  useSelectSelectedToolbarTab: selectSelectedToolbarTab,
  useSelectClickMode: selectClickMode,
  useSelectViewportMode: selectViewportMode,
  useSelectHiddenLines: selectHiddenLines,
  useSelectViewportSelection: selectViewportSelection,
  useSelectSelectedRules: selectSelectedRules,
  useSelectHold: selectHold,
  useSelectEnableTooltip: selectEnableTooltip,
  useSelectMeasureMode: selectMeasureMode,
  useSelectShowRangeAnnotations: selectShowRangeAnnotations,
});

export const {
  useSetSelectedToolbarTab,
  useSetClickMode,
  useSetSelectedRules,
  useSetMeasureMode,
  useSetViewportMode,
  useSelectRule,
  useSetHold,
  useSetTooltipsVisible,
  useSetShowRangeAnnotations,
} = Keyed.createDispatchHandlers(Base.useKey, {
  useSetSelectedToolbarTab: ["tab", setSelectedToolbarTab],
  useSetClickMode: ["mode", setClickMode],
  useSetViewportMode: ["mode", setViewportMode],
  useSetSelectedRules: ["ruleKeys", setSelectedRules],
  useSetMeasureMode: ["mode", setMeasureMode],
  useSetRangeAnnotationsVisible: ["visible", setRangeAnnotationsVisible],
  useSelectRule: ["ruleKeyal", selectRule],
  useSetHold: ["hold", setHold],
  useSetTooltipsVisible: ["visible", setTooltipsVisible],
  useSetShowRangeAnnotations: ["visible", setRangeAnnotationsVisible],
});
