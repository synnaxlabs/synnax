// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Auth } from "@/feature/auth";
import { Version } from "@/feature/version";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Version as PlatformVersion } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  mockUpdater,
  RESOURCE_ID,
} from "@/platform/version/testutil";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import {
  createTestClientWithGrants,
  INCOMPATIBLE_UNLICENSED_STATUS,
  mockLocationReload,
  pinLocationOrigin,
  renderWithStatus,
  type StatusHarness,
} from "@/testutil";

const WORKSPACE = "workspace";
const TITLE = "Console 0.61.0 cannot work with Core 0.60.1";
const CORE_NAME = "Local";
const USERNAME_PLACEHOLDER = "synnax";

const INCOMPATIBLE_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "success",
  message: "Connected",
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    clientVersion: "0.61.0",
    nodeVersion: "0.60.1",
    clientServerCompatible: false,
  },
};

const COMPATIBLE_STATUS: connection.Status = {
  ...INCOMPATIBLE_STATUS,
  details: {
    ...INCOMPATIBLE_STATUS.details,
    nodeVersion: "0.61.0",
    clientServerCompatible: true,
  },
};

const renderGuard = async (
  status: connection.Status,
  secure: boolean = false,
  client: Synnax | null = null,
): Promise<StatusHarness> => {
  const core = createCore(CORE_NAME, { secure });
  return await renderWithStatus(
    <PlatformVersion.Provider>
      <Auth.Guard>
        <Version.Guard>
          <span>{WORKSPACE}</span>
        </Version.Guard>
      </Auth.Guard>
      <Modals.Stack />
    </PlatformVersion.Provider>,
    { client, status, preloadedState: createCoreState([core], core.key) },
  );
};

const getLink = (name: string | RegExp): HTMLElement =>
  screen.getByRole("link", { name });

const { host, port } = CONNECTION_PARAMS;

describe("Version.Guard", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
    mockWindows("main");
    // A dev build version skips the update check.
    mockIPC((cmd) => (cmd === "plugin:app|version" ? "0.0.0" : undefined));
  });

  afterEach(clearUpdater);

  it("should show the mismatch screen in place of the workspace", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    expect(screen.getByText(TITLE)).toBeTruthy();
    expect(
      screen.getByText("The Console and the Core must have the same minor version."),
    ).toBeTruthy();
    expect(screen.queryByText(WORKSPACE)).toBeNull();
  });

  it("should show the mismatch screen while the Core is also unlicensed", async () => {
    await renderGuard(INCOMPATIBLE_UNLICENSED_STATUS);
    expect(screen.getByText(TITLE)).toBeTruthy();
    expect(screen.queryByText(WORKSPACE)).toBeNull();
  });

  it("should lift the screen once the Core reports a matching version", async () => {
    const { setStatus } = await renderGuard(INCOMPATIBLE_STATUS);
    expect(screen.queryByText(WORKSPACE)).toBeNull();
    setStatus(COMPATIBLE_STATUS);
    expect(screen.getByText(WORKSPACE)).toBeTruthy();
    expect(screen.queryByText(TITLE)).toBeNull();
  });

  it("should open the served Console in an external browser", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    const link = getLink(/Open in browser/);
    expect(link.getAttribute("href")).toBe(`http://${host}:${port}`);
    expect(link.getAttribute("target")).toBe("_blank");
  });

  it("should open the served Console over TLS for a secure Core", async () => {
    await renderGuard(INCOMPATIBLE_STATUS, true);
    expect(getLink(/Open in browser/).getAttribute("href")).toBe(
      `https://${host}:${port}`,
    );
  });

  it("should link to the update guide", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    const link = getLink(/Learn how to update/);
    expect(link.getAttribute("href")).toBe(
      "https://docs.synnaxlabs.com/reference/enterprise/upgrade#update-the-console-and-drivers",
    );
    expect(link.getAttribute("target")).toBe("_blank");
  });

  it("should show the Cores list to choose another Core", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    fireEvent.click(findButton("Choose another Core"));
    expect(screen.queryByText(TITLE)).toBeNull();
    expect(await screen.findByText(CORE_NAME)).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should offer the update of the Console that a secure Core serves", async () => {
    mockUpdater({ version: "0.61.0", served: "0.60.2" });
    await renderGuard(INCOMPATIBLE_STATUS, true);
    fireEvent.click(await screen.findByRole("button", { name: "Update Console" }));
    expect(await screen.findByText("Version 0.60.2 available")).toBeTruthy();
  });

  it("should explain that it cannot downgrade over an insecure connection", async () => {
    mockUpdater({ version: "0.61.0", served: "0.60.2" });
    await renderGuard(INCOMPATIBLE_STATUS);
    expect(
      await screen.findByText("Cannot downgrade over an insecure connection"),
    ).toBeTruthy();
    expect(
      screen.getByText(
        "Core 0.60.1 is older than Console 0.61.0. Install Console 0.60.1 by hand, or connect to the Core over TLS to downgrade.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Update Console" })).toBeNull();
  });

  it("should not offer an update when the Core serves none", async () => {
    const ipc = mockUpdater({ version: "0.61.0", served: null });
    await renderGuard(INCOMPATIBLE_STATUS, true);
    await waitFor(() =>
      expect(callsOf(ipc, "updater_check").length).toBeGreaterThan(0),
    );
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "Update Console" })).toBeNull();
  });

  it("should not offer an update that moves the Console past the Core", async () => {
    const granted = await createTestClientWithGrants(createTestClient(), {
      update: [RESOURCE_ID],
    });
    const ipc = mockUpdater({ version: "0.61.0", served: null, latest: "0.61.3" });
    await renderGuard(INCOMPATIBLE_STATUS, true, granted);
    await waitFor(() =>
      expect(callsOf(ipc, "plugin:updater|check").length).toBeGreaterThan(0),
    );
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "Update Console" })).toBeNull();
  });

  it("should render the workspace before a check reports a version", async () => {
    await renderGuard(connection.DEFAULT_STATUS);
    expect(screen.getByText(WORKSPACE)).toBeTruthy();
  });

  describe("in a browser", () => {
    beforeEach(() => {
      mocks.engine = "web";
    });

    it("should reload the page when the Core changes to another minor version", async () => {
      pinLocationOrigin(`http://${host}:${port}`);
      const reload = mockLocationReload();
      const { setStatus } = await renderGuard(COMPATIBLE_STATUS);
      expect(reload).not.toHaveBeenCalled();
      setStatus(INCOMPATIBLE_STATUS);
      expect(reload).toHaveBeenCalledOnce();
      expect(screen.queryByText(TITLE)).toBeNull();
      expect(screen.queryByText(WORKSPACE)).toBeNull();
    });

    it("should show the mismatch screen when another Core served the page", async () => {
      pinLocationOrigin(`http://${host}:${port + 1}`);
      const reload = mockLocationReload();
      await renderGuard(INCOMPATIBLE_STATUS);
      expect(reload).not.toHaveBeenCalled();
      expect(screen.getByText(TITLE)).toBeTruthy();
    });
  });
});
