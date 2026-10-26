// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { emit, TauriEvent } from "@tauri-apps/api/event";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { LogicalPosition } from "@tauri-apps/api/window";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type Event } from "@/runtime";
import { runtimeSetWindowProps, type StoreState } from "@/state";
import { TauriRuntime } from "@/tauri";

const WINDOW_PROPS: Record<string, unknown> = {
  "plugin:window|scale_factor": 1,
  "plugin:window|is_visible": true,
  "plugin:window|is_maximized": false,
  "plugin:window|is_minimized": false,
  "plugin:window|is_fullscreen": true,
  "plugin:window|inner_position": { x: 0, y: 0 },
  "plugin:window|inner_size": { width: 800, height: 600 },
  "plugin:window|available_monitors": [],
};

describe("TauriRuntime", () => {
  let created: Record<string, unknown> | undefined;
  beforeEach(() => {
    created = undefined;
    mockWindows("main");
    mockIPC(
      (cmd, args) => {
        if (cmd === "plugin:webview|create_webview_window")
          created = (args as { options: Record<string, unknown> }).options;
        return WINDOW_PROPS[cmd] ?? null;
      },
      { shouldMockEvents: true },
    );
  });
  afterEach(() => clearMocks());

  describe("create", () => {
    it("should hide the title and place the window controls", async () => {
      const runtime = new TauriRuntime({ trafficLightPosition: { x: 12, y: 23 } });
      await runtime.create("child", {});
      expect(created).toMatchObject({
        label: "child",
        titleBarStyle: "overlay",
        hiddenTitle: true,
        trafficLightPosition: new LogicalPosition(12, 23),
      });
    });

    it("should leave the window controls to Tauri without a position", async () => {
      const runtime = new TauriRuntime();
      await runtime.create("child", {});
      expect(created?.trafficLightPosition).toBeUndefined();
    });
  });

  describe("subscribe", () => {
    it("should report fullscreen when the window resizes", async () => {
      const runtime = new TauriRuntime();
      const listener = vi.fn<(event: Event<StoreState>) => void>();
      await runtime.subscribe(listener);
      await emit(TauriEvent.WINDOW_RESIZED);
      await vi.waitFor(() => expect(listener).toHaveBeenCalled());
      runtime.release();
      const { action } = listener.mock.calls[0][0];
      expect(runtimeSetWindowProps.match(action)).toBe(true);
      expect(action?.payload).toMatchObject({ label: "main", fullscreen: true });
    });
  });
});
