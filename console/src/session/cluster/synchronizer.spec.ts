// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Synnax } from "@synnaxlabs/pluto";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { SYNCHRONIZERS } from "@/session/cluster/synchronizer";
import { createCluster, createClusterState } from "@/session/cluster/testutil";
import { Synchronizer } from "@/session/synchronizer";
import { createConnectedConsoleWrapper, renderHookWithConsole } from "@/testutil";

const retrieveServerClusterKey = async (): Promise<string> => {
  const client = createTestClient();
  const { details } = await client.connect();
  client.close();
  return details.clusterKey;
};

const useSyncWithConnectionState = () => {
  Synchronizer.use(SYNCHRONIZERS);
  return Synnax.useConnectionStatus();
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
    expect(result.current.details.clusterKey).toEqual(serverKey);
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

describe("useCloseOnClusterChange", () => {
  const AUX_LABEL = "aux";
  const AUX_KEY = "aux-key";

  const auxWindow: Drift.WindowState = {
    key: AUX_KEY,
    stage: "created",
    processCount: 0,
    reserved: true,
    focusCount: 0,
    centerCount: 0,
    ordinal: 2,
  };

  const useSyncWithModals = () => {
    const modals = Session.Modals.useStore("useCloseOnClusterChange spec");
    Synchronizer.use(SYNCHRONIZERS);
    return { modals, status: Synnax.useConnectionStatus() };
  };

  it("should leave open windows and modals alone on first connect", async () => {
    const serverKey = await retrieveServerClusterKey();
    const active = createCluster(serverKey);
    const { wrapper, store } = await createConnectedConsoleWrapper({
      client: null,
      connParams: active,
      preloadedState: {
        ...createClusterState([active], serverKey),
        drift: {
          ...Session.ZERO_STATE.drift,
          windows: { ...Session.ZERO_STATE.drift.windows, [AUX_LABEL]: auxWindow },
          labelKeys: { ...Session.ZERO_STATE.drift.labelKeys, [AUX_LABEL]: AUX_KEY },
          keyLabels: { ...Session.ZERO_STATE.drift.keyLabels, [AUX_KEY]: AUX_LABEL },
          nextOrdinal: 3,
        },
      },
    });
    const { result } = renderHook(useSyncWithModals, { wrapper });
    act(() =>
      result.current.modals.push(
        () => null,
        {},
        () => {},
      ),
    );

    await waitFor(() => expect(result.current.status.variant).toBe("success"));
    await waitFor(() =>
      expect(result.current.status.details.epoch).toBeGreaterThanOrEqual(1),
    );

    expect(Drift.selectWindow(store.getState(), AUX_KEY)).not.toBeNull();
    expect(result.current.modals.isAnyOpen()).toBe(true);
  });
});
