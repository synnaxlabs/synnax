// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul } from "@synnaxlabs/lyra/haul";
import { mockWindows } from "@tauri-apps/api/mocks";
import { createEvent, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Window } from "@/app/window";
import { clearUpdater, createStatus, mockUpdater } from "@/platform/version/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  getBySelector,
  HaulProvider,
  installPortalRoot,
  renderWithConsole,
  renderWithStatus,
} from "@/testutil";

// jsdom has no DragEvent, so the transfer is defined on the event testing-library falls
// back to. effectAllowed marks an OS file drag rather than an internal haul.
const fireFileDragOver = (target: HTMLElement): void => {
  const event = createEvent.dragOver(target);
  Object.defineProperty(event, "dataTransfer", { value: { effectAllowed: "all" } });
  fireEvent(target, event);
};

describe("app/window/Window", () => {
  installPortalRoot();

  afterEach(async () => {
    mocks.engine = "web";
    await clearUpdater();
  });

  // The window's own background carries no actions of its own, so Reload Console is the
  // only thing standing between a right click there and a dead menu.
  it("offers Reload Console from the window background", async () => {
    const { container } = await renderWithConsole(
      <HaulProvider>
        <Window.Window />
      </HaulProvider>,
    );
    fireEvent.contextMenu(getBySelector<HTMLElement>(container, ".console-main"));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  // Dropping a file onto the mosaic imports it, but only while the haul carries FILE.
  // Nothing else starts that drag, so losing this makes every file drop a no-op.
  it("hauls a file when an OS file enters the window", async () => {
    const { store, container } = await renderWithConsole(
      <HaulProvider>
        <Window.Window />
      </HaulProvider>,
    );
    fireFileDragOver(getBySelector<HTMLElement>(container, ".console-main"));
    await waitFor(() =>
      expect(store.getState()[Session.Haul.SLICE_NAME].state.items).toEqual([
        Haul.FILE,
      ]),
    );
  });

  // Without the Provider, every update check reads the empty context default.
  it("tells the user about a Console update from the main window", async () => {
    mocks.engine = "tauri";
    mockUpdater({ served: "1.5.1" });
    mockWindows(Drift.MAIN_WINDOW);
    const core = createCore("Local");
    await renderWithStatus(
      <HaulProvider>
        <Window.Window />
      </HaulProvider>,
      {
        client: null,
        status: createStatus("1.5.0"),
        preloadedState: createCoreState([core], core.key),
      },
    );
    // The status and the nav bar badge both say it.
    expect(await screen.findAllByText("Update available")).not.toHaveLength(0);
  });
});
