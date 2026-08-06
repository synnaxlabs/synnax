// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/feature/panel";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

describe("useOpenWindow", () => {
  it("should open a window seeded to show the given panel", async () => {
    const panelKey = uuid.create();
    const { result, store } = await renderHookWithConsole(() => Panel.useOpenWindow());
    act(() => result.current(panelKey));

    await waitFor(() => {
      const state = store.getState();
      const win = Drift.selectWindows(state).find(
        ({ key, reserved }) => key !== Drift.MAIN_WINDOW && reserved,
      );
      expect(win).toBeDefined();
      expect(state.panels.windows[win!.key]?.selected).toEqual(panelKey);
    });
  });

  it("should leave the new window's selection empty when no panel is given", async () => {
    const { result, store } = await renderHookWithConsole(() => Panel.useOpenWindow());
    act(() => result.current());

    await waitFor(() => {
      const state = store.getState();
      const win = Drift.selectWindows(state).find(
        ({ key, reserved }) => key !== Drift.MAIN_WINDOW && reserved,
      );
      expect(win).toBeDefined();
      expect(state.panels.windows[win!.key]).toBeUndefined();
    });
  });

  it("should not disturb the spawning window's selection", async () => {
    const panelKey = uuid.create();
    const original = uuid.create();
    const { result, store } = await renderHookWithConsole(() => Panel.useOpenWindow());
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: original, windowKey: Drift.MAIN_WINDOW }),
      );
      result.current(panelKey);
    });

    await waitFor(() => {
      const state = store.getState();
      expect(state.panels.windows[Drift.MAIN_WINDOW]?.selected).toEqual(original);
    });
  });
});
