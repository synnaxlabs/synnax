// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(
  (): {
    engine: "web" | "tauri";
    update: {
      version: string;
      download: ReturnType<typeof vi.fn>;
      install: ReturnType<typeof vi.fn>;
    } | null;
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

import { Modals } from "@/platform/modals";
import { Wrapper } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Wrapper>
    {children}
    <Modals.Stack />
  </Wrapper>
);

const openModal = (middleware?: Version.InstallMiddleware): void => {
  const { result } = renderHook(Version.useInfoModal, {
    wrapper:
      middleware == null
        ? wrapper
        : ({ children }: PropsWithChildren): ReactElement => (
            <Version.InstallProvider middleware={middleware}>
              {wrapper({ children })}
            </Version.InstallProvider>
          ),
  });
  act(() => {
    result.current();
  });
};

const clickUpdate = async (): Promise<void> => {
  await waitFor(() => expect(screen.getByText("Version 9.9.9 available")).toBeTruthy());
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Update and restart" }));
  });
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
    const download = vi.fn(async (onProgress: (event: unknown) => void) => {
      onProgress({ event: "Started", data: { contentLength: 1000 } });
      onProgress({ event: "Progress", data: { chunkLength: 400 } });
      onProgress({ event: "Progress", data: { chunkLength: 600 } });
      onProgress({ event: "Finished" });
    });
    const install = vi.fn(async () => {});
    mocks.update = { version: "9.9.9", download, install };
    openModal();
    await clickUpdate();
    await waitFor(() => expect(mocks.relaunch).toHaveBeenCalledTimes(1));
    expect(download).toHaveBeenCalledTimes(1);
    expect(install).toHaveBeenCalledTimes(1);
  });

  it("should run the install through the provided middleware", async () => {
    mocks.engine = "tauri";
    const order: string[] = [];
    const download = vi.fn(async () => void order.push("download"));
    const install = vi.fn(async () => void order.push("install"));
    mocks.relaunch.mockImplementation(() => void order.push("relaunch"));
    mocks.update = { version: "9.9.9", download, install };
    openModal(async (run) => {
      order.push("before");
      await run();
      order.push("after");
    });
    await clickUpdate();
    await waitFor(() =>
      expect(order).toEqual(["download", "before", "install", "after", "relaunch"]),
    );
  });

  it("should not relaunch when the middleware rejects", async () => {
    mocks.engine = "tauri";
    const install = vi.fn(async () => {});
    mocks.update = { version: "9.9.9", download: vi.fn(async () => {}), install };
    openModal(async () => await Promise.reject(new Error("the app is busy")));
    await clickUpdate();
    expect(await screen.findByText("Failed to update Console")).toBeTruthy();
    expect(install).not.toHaveBeenCalled();
    expect(mocks.relaunch).not.toHaveBeenCalled();
  });

  it("should report up to date in tauri when the check finds no update", async () => {
    mocks.engine = "tauri";
    openModal();
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
    expect(mocks.relaunch).not.toHaveBeenCalled();
  });

  it("should never install an update without a click", async () => {
    mocks.engine = "tauri";
    const download = vi.fn(async () => {});
    mocks.update = { version: "9.9.9", download, install: vi.fn(async () => {}) };
    openModal();
    await waitFor(() =>
      expect(screen.getByText("Version 9.9.9 available")).toBeTruthy(),
    );
    expect(download).not.toHaveBeenCalled();
    expect(mocks.relaunch).not.toHaveBeenCalled();
  });
});
