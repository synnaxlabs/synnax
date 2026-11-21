// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { project, Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Project } from "@synnaxlabs/pluto";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, screen, waitFor } from "@testing-library/react";
import { assert, describe, expect, it, onTestFinished, vi } from "vitest";

// The full-width top bar is macOS only, so the engine stays the sole variable
// across host platforms, Linux CI included.
await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("macOS");
});

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import type { ReactNode } from "react";

import { Primary } from "@/app/window/Primary";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  renderSuspended,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

const client: Synnax = createTestClient();

const PROJECT_KEY = "f5d0a5c8-5a1e-4b1f-9c0e-0b0c1d2e3f40";
const [CORE_KEY] = Object.keys(Session.Core.ZERO_SLICE_STATE.cores);

const withWorkspace = (projectKey: string = PROJECT_KEY): ConsolePreloadedState => ({
  [Session.Core.SLICE_NAME]: {
    ...Session.Core.ZERO_SLICE_STATE,
    selected: CORE_KEY,
  },
  [Session.Project.SLICE_NAME]: {
    ...Session.Project.ZERO_SLICE_STATE,
    selected: projectKey,
  },
});

const WORKSPACE = ".console-main__workspace";
const RAIL = ".console-nav__bar.lyra--location-left";
const TOP_BAR = ".console-nav__bar.lyra--location-top";

const BASE_TITLE = "Synnax Console";

const setEngine = (engine: "web" | "tauri"): void => {
  mocks.engine = engine;
  if (engine !== "tauri") return;
  // A dev build version skips the update check.
  mockIPC((cmd) => (cmd === "plugin:app|version" ? "0.0.0" : undefined));
  onTestFinished(clearMocks);
};

const renderPrimary = async (engine: "web" | "tauri"): Promise<HTMLElement> => {
  setEngine(engine);
  const { container } = await renderWithConsole(<Primary />, {
    preloadedState: withWorkspace(),
    // No Core is connected, so the panel boundary catches the disconnect.
    onCaughtError: () => {},
  });
  const workspace = container.querySelector<HTMLElement>(WORKSPACE);
  assert(workspace != null);
  return workspace;
};

// Reads the query the tab title reads, so its text shows that the name has arrived.
const ProjectName = ({ projectKey }: { projectKey: project.Key }): ReactNode =>
  Project.useResult({ key: projectKey }).data?.name;

interface Connected {
  proj: project.Project;
  store: TestStore;
}

/** Renders Primary against the test Core with a new project selected. */
const renderConnected = async (engine: "web" | "tauri"): Promise<Connected> => {
  setEngine(engine);
  document.title = BASE_TITLE;
  const proj = await client.projects.create({ name: uniqueName("tab") });
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: withWorkspace(proj.key),
  });
  await renderSuspended(
    <>
      <Primary />
      <ProjectName projectKey={proj.key} />
    </>,
    { wrapper },
  );
  await screen.findAllByText(proj.name);
  return { proj, store };
};

describe("app/window/Primary", () => {
  // A rail placed outside the grid silently loses its inset.
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

  describe("browser tab title", () => {
    it("should start with the name of the selected project", async () => {
      const { proj } = await renderConnected("web");
      expect(document.title).toBe(`${proj.name} - ${BASE_TITLE}`);
    });

    it("should follow a rename of the selected project", async () => {
      const { proj } = await renderConnected("web");
      const name = uniqueName("renamed");
      await client.projects.rename(proj.key, name);
      await waitFor(() => expect(document.title).toBe(`${name} - ${BASE_TITLE}`));
    });

    it("should go back to the base title when the project closes", async () => {
      const { store } = await renderConnected("web");
      act(() => {
        store.dispatch(Session.Project.clearSelected());
      });
      expect(document.title).toBe(BASE_TITLE);
    });

    it("should not change on desktop", async () => {
      await renderConnected("tauri");
      expect(document.title).toBe(BASE_TITLE);
    });
  });
});
