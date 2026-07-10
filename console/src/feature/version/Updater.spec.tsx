// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; update: unknown } => ({
  engine: "web",
  update: null,
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: vi.fn(async () => mocks.update),
}));

import { check } from "@tauri-apps/plugin-updater";

import { Version } from "@/feature/version";
import { renderWithModals } from "@/platform/modals/testutil";
import { renderHookWithConsole } from "@/testutil";

const checkMock = vi.mocked(check);

const spec = (key: string): Status.NotificationSpec => ({
  key,
  name: "Version Update",
  variant: "info",
  message: "Update available",
  description: "",
  time: TimeStamp.now(),
  count: 1,
});

describe("version Updater", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.update = null;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("Notification", () => {
    it("should not match statuses that are not version updates", () => {
      expect(Version.Notification.match(spec("someOtherStatus"))).toBe(false);
    });

    it("should match version update statuses", () => {
      expect(Version.Notification.match(spec("versionUpdate-123"))).toBe(true);
    });

    it("should render an update action for a version update status", () => {
      renderWithModals(
        <Version.Notification status={spec("versionUpdate-123")} silence={vi.fn()} />,
      );
      expect(screen.getByRole("button", { name: "Update" })).toBeTruthy();
    });

    it("should be registered in NOTIFICATIONS", () => {
      expect(Version.NOTIFICATIONS).toContain(Version.Notification);
    });
  });

  describe("useCheckForUpdates", () => {
    it("should not check for updates in the web engine", async () => {
      const { result } = await renderHookWithConsole(() =>
        Version.useCheckForUpdates(),
      );
      await waitFor(() => expect(result.current).toBe(false));
      expect(checkMock).not.toHaveBeenCalled();
    });

    it("should report no update when the check finds none in tauri", async () => {
      mocks.engine = "tauri";
      mocks.update = null;
      const { result } = await renderHookWithConsole(() =>
        Version.useCheckForUpdates(),
      );
      await waitFor(() => expect(checkMock).toHaveBeenCalled());
      expect(result.current).toBe(false);
    });

    it("should report an update available in tauri when one is found", async () => {
      mocks.engine = "tauri";
      mocks.update = { version: "1.2.3" };
      const { result } = await renderHookWithConsole(() =>
        Version.useCheckForUpdates(),
      );
      await waitFor(() => expect(result.current).toBe(true));
    });
  });

  describe("OpenUpdateDialogAction", () => {
    it("should open the version info modal when the Update button is clicked", async () => {
      renderWithModals(<Version.OpenUpdateDialogAction />);
      fireEvent.click(screen.getByRole("button", { name: "Update" }));
      await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    });
  });
});
