// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { assert, describe, expect, it, vi } from "vitest";

// The full-width top bar is macOS only, so pin the OS to keep the engine the sole
// variable across host platforms (Linux CI included).
await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("macOS");
});

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Primary } from "@/app/window/Primary";
import { Session } from "@/session";
import { type ConsolePreloadedState, renderWithConsole } from "@/testutil";

const PROJECT_KEY = "f5d0a5c8-5a1e-4b1f-9c0e-0b0c1d2e3f40";
const [CLUSTER_KEY] = Object.keys(Session.Cluster.ZERO_SLICE_STATE.clusters);

const withWorkspace = (): ConsolePreloadedState => ({
  [Session.Cluster.SLICE_NAME]: {
    ...Session.Cluster.ZERO_SLICE_STATE,
    selected: CLUSTER_KEY,
  },
  [Session.Project.SLICE_NAME]: {
    ...Session.Project.ZERO_SLICE_STATE,
    selected: PROJECT_KEY,
  },
});

const WORKSPACE = ".console-main__workspace";
const RAIL = ".console-nav__bar.pluto--location-left";
const TOP_BAR = ".console-nav__bar.pluto--location-top";

const renderPrimary = async (engine: "web" | "tauri"): Promise<HTMLElement> => {
  mocks.engine = engine;
  const { container } = await renderWithConsole(<Primary />, {
    preloadedState: withWorkspace(),
  });
  const workspace = container.querySelector<HTMLElement>(WORKSPACE);
  assert(workspace != null);
  return workspace;
};

describe("app/window/Primary", () => {
  // The rail's inset comes from the workspace grid, so a rail placed anywhere else
  // silently loses it. Both engines must hand the grid the same children.
  it.each(["web", "tauri"] as const)(
    "should mount the top bar and the rail as workspace grid items on %s",
    async (engine) => {
      const workspace = await renderPrimary(engine);
      expect(workspace.querySelector(`:scope > ${TOP_BAR}`)).not.toBeNull();
      expect(workspace.querySelector(`:scope > ${RAIL}`)).not.toBeNull();
    },
  );

  it("should span the top bar across the window on macOS desktop", async () => {
    const workspace = await renderPrimary("tauri");
    expect(workspace.classList).toContain("console--full-width-top");
  });

  it("should keep the top bar beside the rail in the browser", async () => {
    const workspace = await renderPrimary("web");
    expect(workspace.classList).not.toContain("console--full-width-top");
  });
});
