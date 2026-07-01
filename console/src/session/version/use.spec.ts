// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(
  (): {
    engine: "web" | "tauri";
    connectionState: { nodeVersion?: string } | undefined;
    tauriVersion: string;
  } => ({
    engine: "web",
    connectionState: undefined,
    tauriVersion: "9.9.9",
  }),
);

vi.mock("@/session/runtime", () => ({
  Runtime: {
    get ENGINE() {
      return mocks.engine;
    },
  },
}));

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn(async () => mocks.tauriVersion),
}));

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const Synnax = actual.Synnax as Record<string, unknown>;
  return {
    ...actual,
    Synnax: { ...Synnax, useConnectionState: () => mocks.connectionState },
  };
});

import { Version } from "@/session/version";

describe("Version.use", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.connectionState = undefined;
    mocks.tauriVersion = "9.9.9";
  });

  afterEach(() => vi.clearAllMocks());

  describe("web engine", () => {
    it("should return the connected node's version", () => {
      mocks.connectionState = { nodeVersion: "1.2.3" };
      const { result } = renderHook(() => Version.use());
      expect(result.current).toBe("1.2.3");
    });

    it("should return undefined when there is no connection state", () => {
      mocks.connectionState = undefined;
      const { result } = renderHook(() => Version.use());
      expect(result.current).toBeUndefined();
    });

    it("should return undefined when the connection has no node version", () => {
      mocks.connectionState = {};
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
      mocks.connectionState = { nodeVersion: "should-be-ignored" };
      const { result } = renderHook(() => Version.use());
      await waitFor(() => expect(result.current).toBe("0.42.0"));
    });
  });
});
