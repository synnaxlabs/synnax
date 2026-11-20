// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { emit, TauriEvent } from "@tauri-apps/api/event";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { LogicalPosition } from "@tauri-apps/api/window";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { Event } from "@/runtime";
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
  "plugin:window|get_all_windows": ["main"],
};

describe("TauriRuntime", () => {
  let created: Record<string, unknown> | undefined;
  let commands: string[];
  let createError: string | undefined;
  let responses: Record<string, unknown>;
  beforeEach(() => {
    created = undefined;
    commands = [];
    createError = undefined;
    responses = { ...WINDOW_PROPS };
    mockWindows("main");
    mockIPC(
      (cmd, args) => {
        commands.push(cmd);
        if (cmd === "plugin:webview|create_webview_window" && createError != null)
          throw new Error(createError);
        if (cmd === "plugin:webview|create_webview_window")
          created = (args as { options: Record<string, unknown> }).options;
        return responses[cmd] ?? null;
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

    it("should reject when Tauri fails to create the window", async () => {
      createError = "a webview with label `child` already exists";
      const runtime = new TauriRuntime();
      await expect(runtime.create("child", {})).rejects.toThrow();
    });

    it("should move a window off a HiDPI monitor back to the origin", async () => {
      const area = { position: { x: 0, y: 0 }, size: { width: 2000, height: 1000 } };
      // 2000 physical pixels at a scale of 2 span 1000 logical pixels.
      responses["plugin:window|available_monitors"] = [
        { name: "hidpi", scaleFactor: 2, ...area, workArea: area },
      ];
      const runtime = new TauriRuntime();
      await runtime.create("child", { position: { x: 1500, y: 100 } });
      expect(created).toMatchObject({ x: 0, y: 0 });
    });
  });

  describe("close", () => {
    it("should destroy the window only after the hook before the close", async () => {
      const runtime = new TauriRuntime({
        beforeClose: async (label) => {
          // A timer fires after every pending microtask, so a close that does not
          // wait for the hook destroys the window first.
          await scheduler.nextTask();
          commands.push(`before close ${label}`);
        },
      });
      await runtime.close("main");
      const DESTROY = "plugin:window|destroy";
      expect(
        commands.filter((cmd) => cmd === DESTROY || cmd.startsWith("before close")),
      ).toEqual(["before close main", DESTROY]);
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

    it("should report resizes again after the window becomes resizable", async () => {
      // The Tauri event mock keeps a handler after unlisten and warns when it fires.
      vi.spyOn(console, "warn").mockImplementation(() => {});
      const runtime = new TauriRuntime();
      const listener = vi.fn<(event: Event<StoreState>) => void>();
      await runtime.subscribe(listener);
      await runtime.setResizable(false);
      await runtime.setResizable(true);
      await emit(TauriEvent.WINDOW_RESIZED);
      await vi.waitFor(() => expect(listener).toHaveBeenCalled());
      runtime.release();
    });
  });
});
