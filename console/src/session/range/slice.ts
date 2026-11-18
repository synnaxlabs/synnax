// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { array, narrow, numericTimeRangeZ, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * A range the Core holds. Only the key is kept: the name and time range live on the
 * Core, so a copy here could only go stale.
 */
const persistedStateZ = z.object({
  variant: z.literal("persisted"),
  key: z.string(),
});

export interface PersistedState extends z.infer<typeof persistedStateZ> {}

/** A rolling window of the given span. The Core has no notion of one. */
export interface DynamicState {
  variant: "dynamic";
  key: string;
  name: string;
  span: number;
}

export type State = PersistedState | DynamicState;

/**
 * The rolling windows every session offers. Held in code rather than written into
 * stored state, so a release can revise them and no session can lose one. The list
 * always leads with the window new plots and channels open with.
 */
export const BUILT_IN: DynamicState[] = [
  {
    variant: "dynamic",
    key: "recent",
    name: "Rolling 30s",
    span: Number(TimeSpan.seconds(30)),
  },
  {
    variant: "dynamic",
    key: "rolling1m",
    name: "Rolling 1min",
    span: Number(TimeSpan.minutes(1)),
  },
  {
    variant: "dynamic",
    key: "rolling5m",
    name: "Rolling 5min",
    span: Number(TimeSpan.minutes(5)),
  },
  {
    variant: "dynamic",
    key: "rolling15m",
    name: "Rolling 15min",
    span: Number(TimeSpan.minutes(15)),
  },
  {
    variant: "dynamic",
    key: "rolling30m",
    name: "Rolling 30min",
    span: Number(TimeSpan.minutes(30)),
  },
  {
    variant: "dynamic",
    key: "rolling1h",
    name: "Rolling 1h",
    span: Number(TimeSpan.hours(1)),
  },
  {
    variant: "dynamic",
    key: "rolling6h",
    name: "Rolling 6h",
    span: Number(TimeSpan.hours(6)),
  },
  {
    variant: "dynamic",
    key: "rolling12h",
    name: "Rolling 12h",
    span: Number(TimeSpan.hours(12)),
  },
  {
    variant: "dynamic",
    key: "rolling1d",
    name: "Rolling 1d",
    span: Number(TimeSpan.days(1)),
  },
  {
    variant: "dynamic",
    key: "rolling7d",
    name: "Rolling 7d",
    span: Number(TimeSpan.days(7)),
  },
  {
    variant: "dynamic",
    key: "rolling30d",
    name: "Rolling 30d",
    span: Number(TimeSpan.days(30)),
  },
];

export const groupZ = z.enum(["running", "favorites", "recent"]);

/** A group of the ranges toolbar. */
export type Group = z.infer<typeof groupZ>;

const toolbarStateZ = z.object({ collapsed: z.array(groupZ).default([]) });

const isStatic = (r: unknown): r is Record<string, unknown> =>
  narrow.isObject(r) && r.variant === "static";

// Version 0 could also hold ranges kept only in the Console. Version 1 drops them, and
// keeps their times only while line plots may still hold their keys.
const migrateV0 = (raw: unknown): unknown => {
  if (!narrow.isObject(raw) || (raw.version ?? 0) !== 0) return raw;
  if (!Array.isArray(raw.ranges)) return { ...raw, version: 1 };
  const ranges = raw.ranges.filter((r) => !isStatic(r));
  const staticWindows =
    raw.staticKeysRewritten === true
      ? {}
      : Object.fromEntries(
          raw.ranges.filter(isStatic).map((r) => [r.key, r.timeRange]),
        );
  return { ...raw, version: 1, ranges, staticWindows };
};

export const sliceStateZ = z.preprocess(
  migrateV0,
  z.object({
    version: z.literal(1).default(1),
    /** The Core ranges the session added. The built-ins are not among them. */
    ranges: z.array(persistedStateZ).default([]),
    toolbar: toolbarStateZ.prefault({}),
    /**
     * The times of the ranges that Desktop 1.0 kept only in the Console, by key. Line
     * plots may still hold those keys, which the Core cannot resolve.
     */
    staticWindows: z.record(z.string(), numericTimeRangeZ).default({}),
  }),
);

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export type SliceState = z.infer<typeof sliceStateZ>;

export const SLICE_NAME = "range";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type AddPayload = PersistedState | PersistedState[];

interface RemovePayload {
  keys: string[];
}

export interface SetGroupCollapsedPayload {
  group: Group;
  collapsed: boolean;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (state, { payload: range }: PayloadAction<AddPayload>) => {
      const ranges = array.toArray(range);
      const keys = ranges.map(({ key }) => key);
      state.ranges = [...state.ranges.filter((r) => !keys.includes(r.key)), ...ranges];
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      state.ranges = state.ranges.filter(({ key }) => !keys.includes(key));
    },
    setGroupCollapsed: (
      { toolbar },
      { payload: { group, collapsed } }: PayloadAction<SetGroupCollapsedPayload>,
    ) => {
      toolbar.collapsed = toolbar.collapsed.filter((g) => g !== group);
      if (collapsed) toolbar.collapsed.push(group);
    },
    clearStaticWindows: (state) => {
      state.staticWindows = {};
    },
  },
});
export const { add, remove, setGroupCollapsed, clearStaticWindows } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
