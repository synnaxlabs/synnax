// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import {
  callsOf,
  clearUpdater,
  createWrapper,
  type IPC,
  mockLatestCore,
  mockUpdater,
  requestedLatestCore,
  RESOURCE_ID,
} from "@/platform/version/testutil";
import { awaitGranted, createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

/** Renders a Provider for a Core at coreVersion and returns its notifications. */
const renderNotifications = async (
  granted: boolean,
  coreVersion = "1.5.0",
): Promise<{ current: Status.NotificationSpec[] }> => {
  const user = granted
    ? await createTestClientWithGrants(client, { update: [RESOURCE_ID] })
    : null;
  // The first check then has the grant, so the Provider checks only once.
  if (user != null) await awaitGranted(user, RESOURCE_ID, "update");
  const { result } = renderHook(() => Status.useNotifications().statuses, {
    wrapper: await createWrapper({ client: user, coreVersion }),
  });
  return result;
};

describe("version Core notice", () => {
  let ipc: IPC;
  beforeEach(() => {
    ipc = mockUpdater();
  });

  afterEach(async () => {
    await clearUpdater();
    vi.restoreAllMocks();
  });

  it("should tell a user with the grant about a newer Core", async () => {
    mockLatestCore("1.6.0");
    const notifications = await renderNotifications(true);
    await waitFor(() =>
      expect(notifications.current).toContainEqual(
        expect.objectContaining({
          name: "Core",
          message: "Version 1.6.0 available",
          description:
            "Run synnax update 1.6.0 as an administrator on the computer that runs the Core. In Docker, or if the command fails, follow the upgrade guide at https://docs.synnaxlabs.com/reference/enterprise/upgrade.",
        }),
      ),
    );
  });

  it("should not check for a user without the grant", async () => {
    const fetch = mockLatestCore("1.6.0");
    const notifications = await renderNotifications(false);
    // The Provider checks for a Console update after the Core notice had its chance.
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
    await act(async () => {});
    expect(requestedLatestCore(fetch)).toBe(false);
    expect(notifications.current).toEqual([]);
  });

  it("should not check for a dev Core", async () => {
    const fetch = mockLatestCore("1.6.0");
    const notifications = await renderNotifications(true, "0.0.1");
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
    await act(async () => {});
    expect(requestedLatestCore(fetch)).toBe(false);
    expect(notifications.current).toEqual([]);
  });

  it("should say nothing when the Core is the newest release", async () => {
    const fetch = mockLatestCore("1.5.0");
    const notifications = await renderNotifications(true);
    await waitFor(() => expect(requestedLatestCore(fetch)).toBe(true));
    await act(async () => {});
    expect(notifications.current).toEqual([]);
  });

  it("should say nothing when the docs site cannot be reached", async () => {
    const fetch = mockLatestCore(new Error("offline"));
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const notifications = await renderNotifications(true);
    await waitFor(() => expect(warn).toHaveBeenCalledWith(new Error("offline")));
    expect(requestedLatestCore(fetch)).toBe(true);
    expect(notifications.current).toEqual([]);
  });

  it("should say nothing when the docs site sends an invalid version", async () => {
    mockLatestCore("1.6");
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const notifications = await renderNotifications(true);
    await waitFor(() => expect(warn).toHaveBeenCalledWith(expect.any(z.ZodError)));
    expect(notifications.current).toEqual([]);
  });
});
