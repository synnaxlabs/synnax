// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
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

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Modals } from "@/platform/modals";
import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createStatus,
  createWrapper,
  type CreateWrapperParams,
  type IPC,
  mockLatestCore,
  mockUpdater,
  renderProvider,
  RESOURCE_ID,
} from "@/platform/version/testutil";
import { createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

const openModal = async (params: CreateWrapperParams = {}): Promise<void> => {
  const wrapper = await createWrapper(params);
  const { result } = renderHook(Version.useInfoModal, { wrapper });
  act(() => {
    result.current();
  });
};

const clickUpdate = async (version: string = "1.5.1"): Promise<void> => {
  await waitFor(() =>
    expect(screen.getByText(`Version ${version} available`)).toBeTruthy(),
  );
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Update and restart" }));
  });
};

const installed = (ipc: IPC): boolean =>
  callsOf(ipc, "plugin:updater|install").length > 0;

const restarted = (ipc: IPC): boolean =>
  callsOf(ipc, "plugin:process|restart").length > 0;

type SetStatus = (status: connection.Status) => void;

/**
 * Renders a Provider and its badge for a Core at 1.5.0, then opens the modal from the
 * badge.
 */
const renderBadge = async (client: Client | null = null): Promise<SetStatus> => {
  const setStatus = await renderProvider(
    <>
      <Version.Badge />
      <Modals.Stack />
    </>,
    client,
  );
  fireEvent.click(await screen.findByText("Update available"));
  return setStatus;
};

describe("version useInfoModal", () => {
  let latestCore: MockInstance<typeof fetch>;
  beforeEach(() => {
    mocks.engine = "tauri";
    // The Core notice asks the docs site for the newest Core.
    latestCore = mockLatestCore("1.5.0");
  });

  afterEach(async () => {
    await clearUpdater();
    vi.clearAllMocks();
    latestCore.mockRestore();
  });

  it("should report that the console is up to date in the web engine", async () => {
    mocks.engine = "web";
    await openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
  });

  it("should report up to date on a dev build without checking", async () => {
    const ipc = mockUpdater({ version: "0.0.0", served: "1.5.1" });
    await openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    expect(callsOf(ipc, "updater_check")).toHaveLength(0);
  });

  it("should present the update that the Core serves and install it on request", async () => {
    const ipc = mockUpdater({
      served: "1.5.1",
      download: async (send) => {
        send({ event: "Started", data: { contentLength: 1000 } });
        send({ event: "Progress", data: { chunkLength: 400 } });
        send({ event: "Progress", data: { chunkLength: 600 } });
        send({ event: "Finished" });
      },
    });
    await openModal();
    await clickUpdate();
    await waitFor(() => expect(restarted(ipc)).toBe(true));
    expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(1);
    expect(installed(ipc)).toBe(true);
  });

  it("should run the install through the provided middleware", async () => {
    const ipc = mockUpdater({ served: "1.5.1" });
    const order: string[] = [];
    await openModal({
      installMiddleware: async (run) => {
        order.push(`before, installed: ${installed(ipc)}`);
        await run();
        order.push(`after, installed: ${installed(ipc)}`);
      },
    });
    await clickUpdate();
    await waitFor(() => expect(restarted(ipc)).toBe(true));
    expect(order).toEqual(["before, installed: false", "after, installed: true"]);
  });

  it("should not relaunch when the middleware rejects", async () => {
    const ipc = mockUpdater({ served: "1.5.1" });
    await openModal({
      installMiddleware: async () => await Promise.reject(new Error("the app is busy")),
    });
    await clickUpdate();
    expect(await screen.findByText("Failed to update Console")).toBeTruthy();
    expect(installed(ipc)).toBe(false);
    expect(restarted(ipc)).toBe(false);
  });

  it("should report up to date in tauri when the Core serves no update", async () => {
    const ipc = mockUpdater({ served: null });
    await openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    expect(callsOf(ipc, "updater_check")).toHaveLength(1);
  });

  it("should hold the progress bar empty until the download reports a total", async () => {
    mockUpdater({
      served: "1.5.1",
      download: async () => await new Promise<void>(() => {}),
    });
    await openModal();
    await clickUpdate();
    await waitFor(() => expect(screen.getByText("Downloading update")).toBeTruthy());
    const bar = document.querySelector<HTMLElement>(".lyra-progress-bar");
    expect(bar?.style.width).toEqual("0%");
  });

  it("should never install an update without a click", async () => {
    const ipc = mockUpdater({ served: "1.5.1" });
    await openModal();
    await waitFor(() =>
      expect(screen.getByText("Version 1.5.1 available")).toBeTruthy(),
    );
    expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(0);
    expect(restarted(ipc)).toBe(false);
  });

  describe("when the Core changes during an install", () => {
    const startInstall = async (
      download: Promise<void>,
      recheckFails: boolean = false,
    ): Promise<{ ipc: IPC; rid: unknown }> => {
      const ipc = mockUpdater({
        served: "1.5.1",
        download: async () => await download,
      });
      const setStatus = await renderBadge();
      await clickUpdate();
      await waitFor(() =>
        expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(1),
      );
      const [[, args]] = callsOf(ipc, "plugin:updater|download");
      const { rid } = args as { rid: unknown };
      if (recheckFails) {
        const answer = ipc.getMockImplementation();
        ipc.mockImplementation(async (cmd, cmdArgs) => {
          if (cmd === "updater_check") throw new Error("Core unreachable");
          return await answer?.(cmd, cmdArgs);
        });
      }
      setStatus(createStatus("1.5.1"));
      await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(2));
      return { ipc, rid };
    };

    const closed = (ipc: IPC): unknown[] =>
      callsOf(ipc, "plugin:resources|close").map(
        ([, args]) => (args as { rid: unknown }).rid,
      );

    it("should keep the update open until it installs", async () => {
      let finish = (): void => {};
      const { ipc, rid } = await startInstall(
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
      );
      expect(closed(ipc)).not.toContain(rid);
      finish();
      await waitFor(() => expect(restarted(ipc)).toBe(true));
      expect(callsOf(ipc, "plugin:updater|install")[0][1]).toMatchObject({
        updateRid: rid,
      });
    });

    it("should release the update when the install fails", async () => {
      let fail = (): void => {};
      const { ipc, rid } = await startInstall(
        new Promise<void>((_, reject) => {
          fail = () => reject(new Error("download failed"));
        }),
      );
      fail();
      expect(await screen.findByText("Failed to update Console")).toBeTruthy();
      await waitFor(() => expect(closed(ipc)).toContain(rid));
      expect(installed(ipc)).toBe(false);
    });

    it("should keep the progress when the new check fails", async () => {
      await startInstall(new Promise<void>(() => {}), true);
      await act(async () => {});
      expect(screen.getByText("Downloading update")).toBeTruthy();
      expect(screen.queryByText("Failed to check for updates")).toBeNull();
    });
  });

  describe("older release that the Core serves", () => {
    const DOWNGRADE = "Downgrade Console 1.5.0 to 1.4.2";

    const openDowngrade = async (): Promise<IPC> => {
      const ipc = mockUpdater({ served: "1.4.2" });
      await openModal({ coreVersion: "1.4.2", secure: true });
      await clickUpdate("1.4.2");
      expect(await screen.findByText(DOWNGRADE)).toBeTruthy();
      return ipc;
    };

    it("should install it after the user confirms the downgrade", async () => {
      const ipc = await openDowngrade();
      fireEvent.click(screen.getByRole("button", { name: "Downgrade" }));
      await waitFor(() => expect(restarted(ipc)).toBe(true));
      expect(installed(ipc)).toBe(true);
    });

    it("should not download it when the user cancels", async () => {
      const ipc = await openDowngrade();
      fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
      await waitFor(() => expect(screen.queryByText(DOWNGRADE)).toBeNull());
      await act(async () => {});
      expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(0);
    });

    it("should explain that it cannot downgrade over an insecure connection", async () => {
      mockUpdater({ served: "1.4.2" });
      await openModal({ coreVersion: "1.4.2" });
      expect(
        await screen.findByText("Cannot downgrade over an insecure connection"),
      ).toBeTruthy();
      expect(
        screen.getByText(
          "Core 1.4.2 is older than Console 1.5.0. Install Console 1.4.2 by hand, or connect to the Core over TLS to downgrade.",
        ),
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Update and restart" })).toBeNull();
    });
  });

  describe("newest release past the Core", () => {
    const openBreaking = async (): Promise<IPC> => {
      const granted = await createTestClientWithGrants(client, {
        update: [RESOURCE_ID],
      });
      const ipc = mockUpdater({ served: "1.5.0", latest: "1.6.0" });
      await openModal({ client: granted });
      await clickUpdate("1.6.0");
      expect(
        await screen.findByText("Console 1.6.0 cannot work with Core 1.5.0"),
      ).toBeTruthy();
      return ipc;
    };

    it("should install it after the user confirms the break", async () => {
      const ipc = await openBreaking();
      fireEvent.click(screen.getByRole("button", { name: "Update" }));
      await waitFor(() => expect(restarted(ipc)).toBe(true));
      expect(installed(ipc)).toBe(true);
    });

    it("should not download it when the user cancels", async () => {
      const ipc = await openBreaking();
      fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
      await waitFor(() =>
        expect(
          screen.queryByText("Console 1.6.0 cannot work with Core 1.5.0"),
        ).toBeNull(),
      );
      await act(async () => {});
      expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(0);
    });

    it("should not download it when the Core changes before the user confirms", async () => {
      const granted = await createTestClientWithGrants(client, {
        update: [RESOURCE_ID],
      });
      const ipc = mockUpdater({ served: "1.5.0", latest: "1.6.0" });
      const setStatus = await renderBadge(granted);
      await clickUpdate("1.6.0");
      await screen.findByText("Console 1.6.0 cannot work with Core 1.5.0");
      setStatus(createStatus("1.5.1"));
      await waitFor(() => expect(callsOf(ipc, "plugin:updater|check")).toHaveLength(2));
      fireEvent.click(screen.getByRole("button", { name: "Update" }));
      expect(
        await screen.findByText(
          "The Core changed before the update started. Close this dialog and update again.",
        ),
      ).toBeTruthy();
      expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(0);
    });
  });
});
