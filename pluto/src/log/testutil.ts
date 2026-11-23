// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

/** The log state that the Aether mock reports until a test overrides it. */
export const DEFAULT_STATE = {
  region: { one: { x: 0, y: 0 }, two: { x: 400, y: 500 } },
  wheelPos: 0,
  scrolling: false,
  resumedAt: 0,
  empty: true,
  visible: true,
  channelNamesHidden: false,
  timestampPrecision: 0,
  channelNames: {},
  channels: [],
  telem: { type: "noop-log-source", props: {}, variant: "source", valueType: "log" },
  font: "p",
  color: "#00000000",
  overshoot: { x: 0, y: 0 },
  selectionStart: -1,
  selectionEnd: -1,
  visibleStart: 0,
  computedLineHeight: 16,
  rowEntries: [],
};

const LINES = [
  { text: "<one>", color: "" },
  { text: "two", color: "#ff0000" },
];

/** The `Aether.use` mock's log state, its setter, and the selection spy. */
export interface AetherSetup {
  setState: Mock;
  state: Record<string, unknown>;
  selection: Mock;
}

/**
 * Returns a function that makes the mock of `Aether.use` report the log state with the
 * overrides.
 */
export const createAetherSetup =
  (use: Mock) =>
  (overrides: Record<string, unknown> = {}, partial = false): AetherSetup => {
    const setState = vi.fn();
    const state = { ...DEFAULT_STATE, ...overrides };
    const selection = vi.fn(async () => ({ lines: LINES, partial }));
    use.mockReturnValue(["test-key", state, setState, { selection }]);
    return { setState, state, selection };
  };

/** @returns the initial state of the first log that the `Aether.use` mock mounted. */
export const initialStateOf = (use: Mock): Record<string, unknown> =>
  use.mock.calls[0][0].initialState;
