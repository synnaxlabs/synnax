// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createWrapper,
  type CreateWrapperParams,
  type IPC,
  mockUpdater,
  VERSION_RESOURCE_ID,
} from "@/platform/version/testutil";
import { createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

interface OpenModalParams extends CreateWrapperParams {
  middleware?: Version.InstallMiddleware;
}

const openModal = async ({
  middleware,
  ...params
}: OpenModalParams = {}): Promise<void> => {
  const Base = await createWrapper(params);
  const wrapper =
    middleware == null
      ? Base
      : ({ children }: PropsWithChildren): ReactElement => (
          <Version.InstallProvider middleware={middleware}>
            <Base>{children}</Base>
          </Version.InstallProvider>
        );
  const { result } = renderHook(Version.useInfoModal, { wrapper });
  act(() => {
    result.current();
  });
};

const clickUpdate = async (version: string = "1.4.2"): Promise<void> => {
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

describe("version useInfoModal", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
  });

  afterEach(async () => {
    await clearUpdater();
    vi.clearAllMocks();
  });

  it("should report that the console is up to date in the web engine", async () => {
    mocks.engine = "web";
    await openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
  });

  it("should report up to date on a dev build without checking", async () => {
    const ipc = mockUpdater({ version: "0.0.0", served: "1.4.2" });
    await openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    expect(callsOf(ipc, "updater_check")).toHaveLength(0);
  });

  it("should present the update that the Core serves and install it on request", async () => {
    const ipc = mockUpdater({
      served: "1.4.2",
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
    const ipc = mockUpdater({ served: "1.4.2" });
    const order: string[] = [];
    await openModal({
      middleware: async (run) => {
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
    const ipc = mockUpdater({ served: "1.4.2" });
    await openModal({
      middleware: async () => await Promise.reject(new Error("the app is busy")),
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
      served: "1.4.2",
      download: async () => await new Promise<void>(() => {}),
    });
    await openModal();
    await clickUpdate();
    await waitFor(() => expect(screen.getByText("Downloading update")).toBeTruthy());
    const bar = document.querySelector<HTMLElement>(".lyra-progress-bar");
    expect(bar?.style.width).toEqual("0%");
  });

  it("should never install an update without a click", async () => {
    const ipc = mockUpdater({ served: "1.4.2" });
    await openModal();
    await waitFor(() =>
      expect(screen.getByText("Version 1.4.2 available")).toBeTruthy(),
    );
    expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(0);
    expect(restarted(ipc)).toBe(false);
  });

  describe("newest release past the Core", () => {
    const openBreaking = async (): Promise<IPC> => {
      const granted = await createTestClientWithGrants(client, {
        update: [VERSION_RESOURCE_ID],
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
      fireEvent.click(screen.getByRole("button", { name: "Update Console" }));
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
  });
});
