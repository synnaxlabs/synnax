// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { clearMocks } from "@tauri-apps/api/mocks";
import { fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { renderWithModals } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";
import {
  callsOf,
  createWrapper,
  ENDPOINT,
  mockUpdater,
  VERSION_RESOURCE_ID,
} from "@/platform/version/testutil";
import { createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

const spec = (key: string): Status.NotificationSpec => ({
  key,
  name: "Version Update",
  variant: "info",
  message: "Update available",
  description: "",
  time: TimeStamp.now(),
  count: 1,
});

const renderCheck = async (params?: Parameters<typeof createWrapper>[0]) =>
  renderHook(() => Version.useCheckForUpdates(), {
    wrapper: await createWrapper(params),
  });

describe("version Updater", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
  });

  afterEach(() => {
    clearMocks();
    vi.clearAllMocks();
  });

  describe("Notification", () => {
    it("should not match statuses that are not version updates", () => {
      expect(Version.Notification.match(spec("someOtherStatus"))).toBe(false);
    });

    it("should match version update statuses", () => {
      expect(Version.Notification.match(spec("versionUpdate-123"))).toBe(true);
    });

    it("should open the version info modal from its update action", async () => {
      mocks.engine = "web";
      renderWithModals(
        <Version.Notification status={spec("versionUpdate-123")} silence={vi.fn()} />,
      );
      fireEvent.click(screen.getByRole("button", { name: "Update" }));
      await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    });

    it("should be registered in NOTIFICATIONS", () => {
      expect(Version.NOTIFICATIONS).toContain(Version.Notification);
    });
  });

  describe("useCheckForUpdates", () => {
    it("should not check for updates in the web engine", async () => {
      mocks.engine = "web";
      const ipc = mockUpdater({ served: "1.4.0" });
      const { result } = await renderCheck();
      await waitFor(() => expect(result.current).toBe(false));
      expect(ipc).not.toHaveBeenCalled();
    });

    it("should not check for updates on a dev build", async () => {
      const ipc = mockUpdater({ version: "0.0.0", served: "1.4.0" });
      const { result } = await renderCheck();
      await waitFor(() => expect(callsOf(ipc, "plugin:app|version")).toHaveLength(1));
      expect(callsOf(ipc, "updater_check")).toHaveLength(0);
      expect(result.current).toBe(false);
    });

    it("should not check before a check of the Core reports its version", async () => {
      const ipc = mockUpdater({ served: "1.4.0" });
      const { result } = await renderCheck({ coreVersion: null });
      await waitFor(() => expect(callsOf(ipc, "plugin:app|version")).toHaveLength(1));
      expect(callsOf(ipc, "updater_check")).toHaveLength(0);
      expect(result.current).toBe(false);
    });

    it("should check the manifest of the selected Core", async () => {
      const ipc = mockUpdater({ served: "1.4.0" });
      await renderCheck();
      await waitFor(() =>
        expect(callsOf(ipc, "updater_check")).toEqual([
          ["updater_check", { endpoint: ENDPOINT }],
        ]),
      );
    });

    it("should report no update when the Core serves none", async () => {
      const ipc = mockUpdater({ served: null });
      const { result } = await renderCheck();
      await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
      expect(result.current).toBe(false);
    });

    it("should take an older minor version that the Core serves", async () => {
      mockUpdater({ version: "1.5.0", served: "1.4.2" });
      const { result } = await renderCheck({ coreVersion: "1.4.1" });
      await waitFor(() => expect(result.current).toBe(true));
    });

    it("should take a newer minor version that the Core serves", async () => {
      mockUpdater({ version: "1.5.0", served: "1.6.0" });
      const { result } = await renderCheck({ coreVersion: "1.6.0" });
      await waitFor(() => expect(result.current).toBe(true));
    });

    it("should take a newer patch that the Core serves", async () => {
      mockUpdater({ version: "1.5.0", served: "1.5.1" });
      const { result } = await renderCheck();
      await waitFor(() => expect(result.current).toBe(true));
    });

    it("should skip and release an older patch that the Core serves", async () => {
      const ipc = mockUpdater({ version: "1.5.2", served: "1.5.1" });
      const { result } = await renderCheck();
      await waitFor(() =>
        expect(callsOf(ipc, "plugin:resources|close")).toHaveLength(1),
      );
      expect(result.current).toBe(false);
    });

    it("should keep checking on an interval after a check fails", async () => {
      const ipc = mockUpdater();
      ipc.mockImplementation(async (cmd) => {
        if (cmd === "plugin:app|version") return "1.5.0";
        throw new Error("offline");
      });
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      vi.useFakeTimers();
      try {
        await renderCheck();
        await vi.waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
        await vi.advanceTimersByTimeAsync(TimeSpan.seconds(90).milliseconds);
        expect(callsOf(ipc, "updater_check").length).toBeGreaterThan(1);
      } finally {
        vi.useRealTimers();
        errorSpy.mockRestore();
      }
    });

    describe("newest release", () => {
      it("should offer a release past the minor of the Core to a granted user", async () => {
        const granted = await createTestClientWithGrants(client, {
          update: [VERSION_RESOURCE_ID],
        });
        mockUpdater({ served: "1.5.0", latest: "1.6.0" });
        const { result } = await renderCheck({ client: granted });
        await waitFor(() => expect(result.current).toBe(true));
      });

      it("should not offer a release within the minor of the Core", async () => {
        const granted = await createTestClientWithGrants(client, {
          update: [VERSION_RESOURCE_ID],
        });
        const ipc = mockUpdater({ served: "1.5.0", latest: "1.5.3" });
        const { result } = await renderCheck({ client: granted });
        await waitFor(() =>
          expect(callsOf(ipc, "plugin:resources|close")).toHaveLength(1),
        );
        expect(result.current).toBe(false);
      });

      it("should not check the newest release for a user without the grant", async () => {
        const denied = await createTestClientWithGrants(client);
        const ipc = mockUpdater({ served: "1.5.0", latest: "1.6.0" });
        const { result } = await renderCheck({ client: denied });
        await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
        expect(callsOf(ipc, "plugin:updater|check")).toHaveLength(0);
        expect(result.current).toBe(false);
      });
    });
  });
});
