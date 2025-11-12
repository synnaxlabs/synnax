// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Cluster } from "@/cluster";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Cluster queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useConnectionState", () => {
    it("should retrieve connection state for valid connection", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.status).toBeDefined();
      expect(result.current.data?.clusterKey).toBeDefined();
      expect(result.current.data?.clientVersion).toBeDefined();
    });

    it("should return all expected connection state properties", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
            name: "test-cluster",
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      expect(state).toBeDefined();
      expect(state?.status).toMatch(/disconnected|connecting|connected|failed/);
      expect(typeof state?.clusterKey).toBe("string");
      expect(typeof state?.clientVersion).toBe("string");
      expect(typeof state?.clientServerCompatible).toBe("boolean");
    });

    it("should handle connection state for connected cluster", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      if (state?.status === "connected") {
        expect(state.clusterKey).toBeTruthy();
        expect(state.message).toBeDefined();
      }
    });

    it("should handle invalid connection parameters gracefully", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "invalid-host-that-does-not-exist",
            port: 99999,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      expect(state).toBeDefined();
      expect(["failed", "disconnected"]).toContain(state?.status);
    });

    it("should include error information when connection fails", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "unreachable-host",
            port: 12345,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      if (state?.status === "failed")
        expect(state.error || state.message).toBeDefined();
    });

    it("should work with custom cluster name", async () => {
      const customName = "my-custom-cluster";
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
            name: customName,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.status).toBeDefined();
    });

    it("should allow disconnected queries without error", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "offline-server",
            port: 1234,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.variant).not.toEqual("error");
    });

    it("should handle secure connection parameter", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: true,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.status).toBeDefined();
    });

    it("should properly unmount and cleanup connection checker", async () => {
      const { result, unmount } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();

      unmount();
    });

    it("should maintain state consistency across multiple checks", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const firstState = result.current.data;
      expect(firstState).toBeDefined();

      await waitFor(() => expect(result.current.data).toBeDefined());

      const secondState = result.current.data;
      expect(secondState?.status).toBe(firstState?.status);
      expect(secondState?.clusterKey).toBe(firstState?.clusterKey);
    });

    it("should handle multiple simultaneous connection state queries", async () => {
      const { result: result1 } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      const { result: result2 } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
            name: "second-connection",
          }),
        { wrapper },
      );

      await waitFor(() => expect(result1.current.variant).toEqual("success"));
      await waitFor(() => expect(result2.current.variant).toEqual("success"));

      expect(result1.current.data).toBeDefined();
      expect(result2.current.data).toBeDefined();
      expect(result1.current.data?.status).toBeDefined();
      expect(result2.current.data?.status).toBeDefined();
    });

    it("should include client version in connection state", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      expect(state?.clientVersion).toBeDefined();
      expect(typeof state?.clientVersion).toBe("string");
      expect(state?.clientVersion.length).toBeGreaterThan(0);
    });

    it("should handle client-server compatibility check", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      expect(typeof state?.clientServerCompatible).toBe("boolean");
    });

    it("should provide node version when connected", async () => {
      const { result } = renderHook(
        () =>
          Cluster.useConnectionState({
            host: "localhost",
            port: 9090,
            secure: false,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const state = result.current.data;
      if (state?.status === "connected") expect(state.nodeVersion).toBeDefined();
    });
  });
});
