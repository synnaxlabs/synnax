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
import { screen, waitFor } from "@testing-library/react";
import { isValidElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; update: unknown } => ({
  engine: "web",
  update: null,
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    get ENGINE() {
      return mocks.engine;
    },
  };
});

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: vi.fn(async () => mocks.update),
}));

import { renderWithModals } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";
import { renderHookWithConsole } from "@/testutil";

const spec = (key: string): Status.NotificationSpec =>
  ({
    key,
    variant: "info",
    message: "Update available",
    time: TimeStamp.now(),
    count: 1,
  }) as Status.NotificationSpec;

describe("version Updater", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.update = null;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("notificationAdapter", () => {
    it("should ignore statuses that are not version updates", () => {
      expect(Version.notificationAdapter(spec("someOtherStatus"), vi.fn())).toBeNull();
    });

    it("should attach an update action to a version update status", () => {
      const result = Version.notificationAdapter(spec("versionUpdate-123"), vi.fn());
      expect(result).not.toBeNull();
      expect(result?.key).toEqual("versionUpdate-123");
      const actions = result?.actions;
      expect(Array.isArray(actions)).toBe(true);
      expect(isValidElement((actions as unknown[])[0])).toBe(true);
    });

    it("should be registered in NOTIFICATION_ADAPTERS", () => {
      expect(Version.NOTIFICATION_ADAPTERS).toContain(Version.notificationAdapter);
    });
  });

  describe("useCheckForUpdates", () => {
    it("should never report an update available in the web engine", async () => {
      const { result } = await renderHookWithConsole(() =>
        Version.useCheckForUpdates(),
      );
      await waitFor(() => expect(result.current).toBe(false));
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
    it("should render an Update button", () => {
      renderWithModals(<Version.OpenUpdateDialogAction />);
      expect(screen.getByRole("button", { name: "Update" })).toBeTruthy();
    });
  });
});
