// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Cluster } from "@/platform/cluster";
import { createCluster, createClusterState } from "@/platform/cluster/testutil";
import { Session } from "@/session";
import { createConnectedConsoleWrapper, renderHookWithConsole } from "@/testutil";

const retrieveServerClusterKey = async (): Promise<string> => {
  const client = createTestClient();
  const { clusterKey } = await client.connect();
  client.close();
  return clusterKey;
};

const useSyncWithConnectionState = () => {
  Cluster.useSyncClusterKey();
  return Synnax.useConnectionState();
};

describe("useSyncClusterKey", () => {
  it("should not change the key when the cluster is not connected", async () => {
    const { result, store } = await renderHookWithConsole(useSyncWithConnectionState, {
      preloadedState: createClusterState([createCluster("active-1")], "active-1"),
    });
    expect(result.current.variant).toBe("disabled");
    expect(Session.Cluster.selectState(store.getState(), "active-1")).toBeDefined();
    expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual("active-1");
  });

  it("should not change the key when it already matches the server", async () => {
    const serverKey = await retrieveServerClusterKey();
    const active = createCluster(serverKey);
    const { wrapper, store } = await createConnectedConsoleWrapper({
      client: null,
      connParams: active,
      preloadedState: createClusterState([active], serverKey),
    });
    const { result } = renderHook(useSyncWithConnectionState, { wrapper });
    await waitFor(() => expect(result.current.variant).toBe("success"));
    expect(result.current.clusterKey).toEqual(serverKey);
    expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual(serverKey);
    expect(Session.Cluster.selectState(store.getState(), serverKey)).toBeDefined();
  });

  it("should adopt the server's cluster key once connected", async () => {
    const serverKey = await retrieveServerClusterKey();
    const active = createCluster("active-1");
    const { wrapper, store } = await createConnectedConsoleWrapper({
      client: null,
      connParams: active,
      preloadedState: createClusterState([active], "active-1"),
    });
    renderHook(useSyncWithConnectionState, { wrapper });
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual(serverKey),
    );
    expect(Session.Cluster.selectState(store.getState(), "active-1")).toBeUndefined();
    expect(Session.Cluster.selectState(store.getState(), serverKey)).toBeDefined();
  });
});
