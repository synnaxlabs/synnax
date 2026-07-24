// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; tauriVersion: string } => ({
  engine: "web",
  tauriVersion: "9.9.9",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn(async () => mocks.tauriVersion),
}));

import { Version } from "@/session/version";
import { createConnectedConsoleWrapper } from "@/testutil";

const retrieveNodeVersion = async (): Promise<string> => {
  const client = createTestClient();
  const { nodeVersion } = await client.connect();
  client.close();
  if (nodeVersion == null) throw new Error("cluster did not report a node version");
  return nodeVersion;
};

describe("Version.use", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.tauriVersion = "9.9.9";
  });

  afterEach(() => vi.clearAllMocks());

  describe("web engine", () => {
    it("should return the connected node's version", async () => {
      const nodeVersion = await retrieveNodeVersion();
      const { wrapper } = await createConnectedConsoleWrapper({
        client: null,
        connParams: TEST_CLIENT_PARAMS,
      });
      const { result } = renderHook(() => Version.use(), { wrapper });
      await waitFor(() => expect(result.current).toBe(nodeVersion));
    });

    it("should return undefined when no cluster is connected", () => {
      const { result } = renderHook(() => Version.use());
      expect(result.current).toBeUndefined();
    });
  });

  describe("tauri engine", () => {
    it("should resolve the app version via the tauri api", async () => {
      mocks.engine = "tauri";
      mocks.tauriVersion = "0.42.0";
      const { result } = renderHook(() => Version.use());
      await waitFor(() => expect(result.current).toBe("0.42.0"));
    });

    it("should ignore the connection state's node version in tauri mode", async () => {
      mocks.engine = "tauri";
      mocks.tauriVersion = "0.42.0";
      const nodeVersion = await retrieveNodeVersion();
      const { wrapper } = await createConnectedConsoleWrapper({
        client: null,
        connParams: TEST_CLIENT_PARAMS,
      });
      const { result } = renderHook(
        () => ({ version: Version.use(), connection: Synnax.useConnectionState() }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.connection.variant).toBe("success"));
      expect(result.current.connection.nodeVersion).toBe(nodeVersion);
      await waitFor(() => expect(result.current.version).toBe("0.42.0"));
    });
  });
});
