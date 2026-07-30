// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(
  (): {
    engine: "web" | "tauri";
    update: { version: string; downloadAndInstall: ReturnType<typeof vi.fn> } | null;
    relaunch: ReturnType<typeof vi.fn>;
  } => ({
    engine: "web",
    update: null,
    relaunch: vi.fn(async () => {}),
  }),
);

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: vi.fn(async () => mocks.update),
}));
vi.mock("@tauri-apps/plugin-process", () => ({ relaunch: mocks.relaunch }));

import { Version } from "@/platform/version";
import { renderWithModals } from "@/platform/modals/testutil";

const Harness = (): ReactElement => {
  const open = Version.useInfoModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

const openModal = (): void => {
  renderWithModals(<Harness />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
};

describe("version useInfoModal", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.update = null;
    mocks.relaunch.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should report that the console is up to date in the web engine", async () => {
    openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
  });

  it("should present an available update and install it on request", async () => {
    mocks.engine = "tauri";
    const downloadAndInstall = vi.fn(async (onProgress: (event: unknown) => void) => {
      onProgress({ event: "Started", data: { contentLength: 1000 } });
      onProgress({ event: "Progress", data: { chunkLength: 400 } });
      onProgress({ event: "Progress", data: { chunkLength: 600 } });
      onProgress({ event: "Finished" });
    });
    mocks.update = { version: "9.9.9", downloadAndInstall };
    openModal();
    await waitFor(() =>
      expect(screen.getByText("Version 9.9.9 available")).toBeTruthy(),
    );
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Update & Restart" }));
    });
    await waitFor(() => expect(downloadAndInstall).toHaveBeenCalledTimes(1));
    expect(mocks.relaunch).toHaveBeenCalledTimes(1);
  });
});
