// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { restoreWindows, type SliceState, ZERO_SLICE_STATE } from "@/state";
import {
  INITIAL_PRERENDER_WINDOW_STATE,
  INITIAL_WINDOW_STATE,
  MAIN_WINDOW,
  type WindowState,
} from "@/window";

const reserved = (key: string, props: Partial<WindowState> = {}): WindowState => ({
  ...INITIAL_WINDOW_STATE,
  key,
  reserved: true,
  ...props,
});

const sliceState = (windows: Record<string, WindowState>): SliceState => {
  const labelKeys: Record<string, string> = {};
  const keyLabels: Record<string, string> = {};
  Object.entries(windows).forEach(([label, win]) => {
    if (!win.reserved) return;
    labelKeys[label] = win.key;
    keyLabels[win.key] = label;
  });
  return { ...ZERO_SLICE_STATE, windows, labelKeys, keyLabels };
};

describe("restoreWindows", () => {
  it("should keep the live main window instead of the stored one", () => {
    const live = reserved(MAIN_WINDOW, { size: { width: 100, height: 100 } });
    const next = restoreWindows(
      sliceState({ [MAIN_WINDOW]: live }),
      sliceState({
        [MAIN_WINDOW]: reserved(MAIN_WINDOW, { size: { width: 900, height: 900 } }),
      }),
    );
    expect(next.windows[MAIN_WINDOW]).toEqual(live);
  });

  it("should replace the live secondary windows with the stored ones", () => {
    const next = restoreWindows(
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW), outgoing: reserved("old") }),
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW), incoming: reserved("new") }),
    );
    expect(next.windows.outgoing).toBeUndefined();
    expect(next.windows.incoming?.key).toEqual("new");
  });

  it("should zero the counters on restored windows", () => {
    const next = restoreWindows(
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW) }),
      sliceState({
        incoming: reserved("new", { focusCount: 3, centerCount: 2, processCount: 1 }),
      }),
    );
    expect(next.windows.incoming).toMatchObject({
      focusCount: 0,
      centerCount: 0,
      processCount: 0,
    });
  });

  it("should keep unused pre-rendered windows out of the swap", () => {
    const spare = { ...INITIAL_PRERENDER_WINDOW_STATE };
    const next = restoreWindows(
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW), spare }),
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW) }),
    );
    expect(next.windows.spare).toEqual(spare);
    expect(next.labelKeys.spare).toBeUndefined();
    expect(next.keyLabels[spare.key]).toBeUndefined();
  });

  it("should rebuild the label maps from the merged windows", () => {
    const next = restoreWindows(
      sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW), outgoing: reserved("old") }),
      sliceState({ incoming: reserved("new") }),
    );
    expect(next.labelKeys).toEqual({ [MAIN_WINDOW]: MAIN_WINDOW, incoming: "new" });
    expect(next.keyLabels).toEqual({ [MAIN_WINDOW]: MAIN_WINDOW, new: "incoming" });
  });

  it("should keep the live label and config", () => {
    const current: SliceState = {
      ...sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW) }),
      label: "live-label",
      config: { enablePrerender: false, debug: true, defaultWindowProps: {} },
    };
    const next = restoreWindows(current, {
      ...sliceState({ [MAIN_WINDOW]: reserved(MAIN_WINDOW) }),
      label: "stored-label",
      config: { enablePrerender: true, debug: false, defaultWindowProps: {} },
    });
    expect(next.label).toEqual("live-label");
    expect(next.config).toEqual(current.config);
  });
});
