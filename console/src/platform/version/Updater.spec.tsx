// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; coreEmbedded?: boolean } => ({
  engine: "tauri",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { UncappedFeed } from "@/platform/notifications/testutil";
import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createStatus,
  createWrapper,
  type CreateWrapperParams,
  ENDPOINT,
  mockLatestCore,
  mockUpdater,
  renderProvider,
  RESOURCE_ID,
} from "@/platform/version/testutil";
import { awaitGranted, createTestClientWithGrants, getIconButton } from "@/testutil";

const client = createTestClient();

const NONE: Version.Check = { variant: "success", available: null };

const renderCheck = async (params?: CreateWrapperParams) =>
  renderHook(() => Version.useCheck(), { wrapper: await createWrapper(params) });

const versionOf = (check: Version.Check): string | null =>
  check.variant === "success" ? (check.available?.update.version ?? null) : null;

describe("version Provider", () => {
  let latestCore: MockInstance<typeof fetch>;
  beforeEach(() => {
    mocks.engine = "tauri";
    mocks.coreEmbedded = false;
    // The Core notice asks the docs site for the newest Core.
    latestCore = mockLatestCore("1.5.0");
  });

  afterEach(async () => {
    await clearUpdater();
    vi.clearAllMocks();
    latestCore.mockRestore();
  });

  it("should not check for updates in the web engine", async () => {
    mocks.engine = "web";
    const ipc = mockUpdater({ served: "1.4.0" });
    const { result } = await renderCheck();
    await waitFor(() => expect(result.current).toEqual(NONE));
    expect(ipc).not.toHaveBeenCalled();
  });

  it("should not check for updates on a dev build", async () => {
    const ipc = mockUpdater({ version: "0.0.0", served: "1.4.0" });
    const { result } = await renderCheck();
    await waitFor(() => expect(result.current).toEqual(NONE));
    expect(callsOf(ipc, "updater_check")).toHaveLength(0);
  });

  it("should not check before a check of the Core reports its version", async () => {
    const ipc = mockUpdater({ served: "1.4.0" });
    const { result } = await renderCheck({ coreVersion: null });
    await waitFor(() => expect(result.current).toEqual(NONE));
    expect(callsOf(ipc, "updater_check")).toHaveLength(0);
  });

  it("should find a Synnax Desktop update on the docs site without a Core", async () => {
    mocks.coreEmbedded = true;
    const ipc = mockUpdater({ served: "1.6.0", latest: "1.5.3" });
    const { result } = await renderCheck({ coreVersion: null });
    await waitFor(() => expect(versionOf(result.current)).toBe("1.5.3"));
    expect(result.current).toMatchObject({ available: { variant: "upgrade" } });
    expect(callsOf(ipc, "updater_check")).toHaveLength(0);
  });

  it("should check the manifest of the selected Core at its version", async () => {
    const ipc = mockUpdater({ served: "1.4.0" });
    await renderCheck({ coreVersion: "1.4.2" });
    await waitFor(() =>
      expect(callsOf(ipc, "updater_check")).toEqual([
        ["updater_check", { endpoint: ENDPOINT, coreVersion: "1.4.2", secure: false }],
      ]),
    );
  });

  it("should report no update when the Core serves none", async () => {
    const ipc = mockUpdater({ served: null });
    const { result } = await renderCheck();
    await waitFor(() => expect(result.current).toEqual(NONE));
    expect(callsOf(ipc, "updater_check")).toHaveLength(1);
  });

  it("should offer an older release over TLS as a downgrade", async () => {
    mockUpdater({ version: "1.5.0", served: "1.4.2" });
    const { result } = await renderCheck({ coreVersion: "1.4.1", secure: true });
    await waitFor(() => expect(versionOf(result.current)).toBe("1.4.2"));
    expect(result.current).toMatchObject({ available: { variant: "downgrade" } });
  });

  it("should block a downgrade over an insecure connection", async () => {
    const ipc = mockUpdater({ version: "1.5.0", served: "1.4.2" });
    const { result } = await renderCheck({ coreVersion: "1.4.1" });
    await waitFor(() =>
      expect(result.current).toMatchObject({
        variant: "blocked",
        status: {
          variant: "warning",
          name: "Console",
          message: "Cannot downgrade over an insecure connection",
        },
      }),
    );
    expect(callsOf(ipc, "updater_check")).toEqual([
      ["updater_check", { endpoint: ENDPOINT, coreVersion: "1.4.1", secure: false }],
    ]);
  });

  it("should not block a Core at an older patch over an insecure connection", async () => {
    mockUpdater({ version: "1.5.2", served: null });
    const { result } = await renderCheck({ coreVersion: "1.5.0" });
    await waitFor(() => expect(result.current).toEqual(NONE));
  });

  it("should take a newer minor version that the Core serves", async () => {
    mockUpdater({ version: "1.5.0", served: "1.6.0" });
    const { result } = await renderCheck({ coreVersion: "1.6.0" });
    await waitFor(() => expect(versionOf(result.current)).toBe("1.6.0"));
  });

  it("should offer a newer patch that the Core serves as an upgrade", async () => {
    mockUpdater({ version: "1.5.0", served: "1.5.1" });
    const { result } = await renderCheck();
    await waitFor(() => expect(versionOf(result.current)).toBe("1.5.1"));
    expect(result.current).toMatchObject({ available: { variant: "upgrade" } });
  });

  it("should release the found update when it unmounts", async () => {
    const ipc = mockUpdater({ served: "1.5.1" });
    const { result, unmount } = await renderCheck();
    await waitFor(() => expect(versionOf(result.current)).toBe("1.5.1"));
    unmount();
    await waitFor(() => expect(callsOf(ipc, "plugin:resources|close")).toHaveLength(1));
  });

  it("should report a failed check and keep checking", async () => {
    const ipc = mockUpdater();
    ipc.mockImplementation(async (cmd) => {
      if (cmd === "plugin:app|version") return "1.5.0";
      throw new Error("offline");
    });
    vi.useFakeTimers();
    try {
      const { result } = await renderCheck();
      await act(async () => {
        await vi.advanceTimersByTimeAsync(TimeSpan.seconds(1).milliseconds);
      });
      expect(result.current.variant).toBe("error");
      const first = callsOf(ipc, "updater_check").length;
      await act(async () => {
        await vi.advanceTimersByTimeAsync(TimeSpan.seconds(90).milliseconds);
      });
      expect(callsOf(ipc, "updater_check").length).toBeGreaterThan(first);
    } finally {
      vi.useRealTimers();
    }
  });

  it("should check again when the Core reports another version", async () => {
    const before = mockUpdater({ served: "1.5.0" });
    const setStatus = await renderProvider(<UncappedFeed />);
    await waitFor(() => expect(callsOf(before, "updater_check")).toHaveLength(1));
    await act(async () => {});
    expect(screen.queryByText("Update available")).toBeNull();
    mockUpdater({ served: "1.6.0" });
    setStatus(createStatus("1.6.0"));
    expect(await screen.findByText("Update available")).toBeTruthy();
  });

  it("should keep a silenced update silent when a new check finds it again", async () => {
    const ipc = mockUpdater({ served: "1.5.1" });
    const setStatus = await renderProvider(<UncappedFeed />);
    await screen.findByText("Update available");
    fireEvent.click(getIconButton(document.body, "close"));
    await waitFor(() => expect(screen.queryByText("Update available")).toBeNull());
    setStatus(createStatus("1.5.2"));
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(2));
    await act(async () => {});
    expect(screen.queryByText("Update available")).toBeNull();
  });

  describe("newest release", () => {
    const createGranted = async () =>
      await createTestClientWithGrants(client, { update: [RESOURCE_ID] });

    it("should offer a release past the minor of the Core to a granted user", async () => {
      mockUpdater({ served: "1.5.0", latest: "1.6.0" });
      const { result } = await renderCheck({ client: await createGranted() });
      await waitFor(() =>
        expect(result.current).toMatchObject({
          variant: "success",
          available: { variant: "breaking", coreVersion: "1.5.0" },
        }),
      );
      expect(versionOf(result.current)).toBe("1.6.0");
    });

    it("should not offer a release within the minor of the Core", async () => {
      const ipc = mockUpdater({ served: "1.5.0", latest: "1.5.3" });
      const { result } = await renderCheck({ client: await createGranted() });
      await waitFor(() =>
        expect(callsOf(ipc, "plugin:resources|close")).toHaveLength(1),
      );
      await waitFor(() => expect(result.current).toEqual(NONE));
    });

    it("should treat an unreachable docs site as no newer release", async () => {
      const ipc = mockUpdater({ served: "1.5.0" });
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      ipc.mockImplementation(async (cmd) => {
        if (cmd === "plugin:app|version") return "1.5.0";
        if (cmd === "plugin:updater|check") throw new Error("offline");
        return null;
      });
      try {
        const { result } = await renderCheck({ client: await createGranted() });
        await waitFor(() =>
          expect(callsOf(ipc, "plugin:updater|check")).toHaveLength(1),
        );
        await waitFor(() => expect(result.current).toEqual(NONE));
      } finally {
        warn.mockRestore();
      }
    });

    it("should not check the newest release against a dev Core", async () => {
      const ipc = mockUpdater({ served: "1.5.0", latest: "1.6.0" });
      const granted = await createGranted();
      await awaitGranted(granted, RESOURCE_ID, "update");
      const { result } = await renderCheck({ client: granted, coreVersion: "0.0.0" });
      await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
      await waitFor(() => expect(result.current).toEqual(NONE));
      expect(callsOf(ipc, "plugin:updater|check")).toHaveLength(0);
    });

    it("should not check the newest release for a user without the grant", async () => {
      const denied = await createTestClientWithGrants(client);
      const ipc = mockUpdater({ served: "1.5.0", latest: "1.6.0" });
      const { result } = await renderCheck({ client: denied });
      await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
      await waitFor(() => expect(result.current).toEqual(NONE));
      expect(callsOf(ipc, "plugin:updater|check")).toHaveLength(0);
    });
  });
});
