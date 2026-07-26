// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Synchronizer } from "@/session/synchronizer";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("Synchronizer.use", () => {
  it("should run reconciles sequentially in declaration order", async () => {
    const calls: string[] = [];
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const { wrapper } = await createConsoleWrapper({ client });
    renderHook(
      () =>
        Synchronizer.use({
          useFirst: () => ({
            reconcile: async () => {
              calls.push("first:start");
              await gate;
              calls.push("first:end");
            },
          }),
          useSecond: () => ({
            reconcile: () => {
              calls.push("second");
            },
          }),
        }),
      { wrapper },
    );
    await waitFor(() => expect(calls).toContain("first:start"));
    expect(calls).not.toContain("second");
    release();
    await waitFor(() => expect(calls).toContain("second"));
    expect(calls).toEqual(["first:start", "first:end", "second"]);
  });

  it("should keep reconciling after an earlier synchronizer fails", async () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const calls: string[] = [];
      const { wrapper } = await createConsoleWrapper({ client });
      renderHook(
        () =>
          Synchronizer.use({
            useBroken: () => ({
              reconcile: () => {
                throw new Error("broken");
              },
            }),
            useHealthy: () => ({
              reconcile: () => {
                calls.push("healthy");
              },
            }),
          }),
        { wrapper },
      );
      await waitFor(() => expect(calls).toContain("healthy"));
      expect(error).toHaveBeenCalledWith(
        "useBroken reconcile failed",
        expect.any(Error),
      );
    } finally {
      error.mockRestore();
    }
  });

  it("should mount listeners before the first reconcile and tear down on unmount", async () => {
    const calls: string[] = [];
    let destroyed = false;
    const { wrapper } = await createConsoleWrapper({ client });
    const { unmount } = renderHook(
      () =>
        Synchronizer.use({
          useListener: () => ({
            reconcile: () => {
              calls.push("reconcile");
            },
            listen: () => {
              calls.push("listen");
              return () => {
                destroyed = true;
              };
            },
          }),
        }),
      { wrapper },
    );
    await waitFor(() => expect(calls).toContain("reconcile"));
    expect(calls[0]).toEqual("listen");
    unmount();
    expect(destroyed).toBe(true);
  });

  it("should report verified only after a full pass completes", async () => {
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(
      () =>
        Synchronizer.use({
          useSlow: () => ({
            reconcile: async () => {
              await gate;
            },
          }),
        }),
      { wrapper },
    );
    expect(result.current).toBe(false);
    release();
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should reset verified when the epoch returns to 0", async () => {
    const local = createTestClient();
    try {
      const { wrapper } = await createConsoleWrapper({ client: local });
      const { result } = renderHook(
        () => Synchronizer.use({ useNoop: () => ({ reconcile: () => {} }) }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBe(true));
      await local.cache.reset();
      await waitFor(() => expect(result.current).toBe(false));
      await local.cache.ensureStreaming();
      await waitFor(() => expect(result.current).toBe(true));
    } finally {
      local.close();
    }
  });

  it("should not invoke anything without a client", async () => {
    const calls: string[] = [];
    const { wrapper } = await createConsoleWrapper({ client: null });
    renderHook(
      () =>
        Synchronizer.use({
          useIdle: () => ({
            reconcile: () => {
              calls.push("reconcile");
            },
            listen: () => {
              calls.push("listen");
              return () => {};
            },
          }),
        }),
      { wrapper },
    );
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(calls).toEqual([]);
  });
});
