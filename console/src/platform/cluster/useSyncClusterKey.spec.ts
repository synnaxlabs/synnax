// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { connectionState: Partial<connection.State> } => ({
  connectionState: {},
}));

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const Synnax = actual.Synnax as Record<string, unknown>;
  return {
    ...actual,
    Synnax: { ...Synnax, useConnectionState: () => mocks.connectionState },
  };
});

import { Cluster } from "@/platform/cluster";
import { cluster, clusterState } from "@/platform/cluster/testutil";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

describe("useSyncClusterKey", () => {
  beforeEach(() => {
    mocks.connectionState = {};
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should not change the key when the cluster is not connected", async () => {
    mocks.connectionState = { status: "disconnected", clusterKey: "server-key" };
    const { store } = await renderHookWithConsole(() => Cluster.useSyncClusterKey(), {
      preloadedState: clusterState([cluster("active-1")], "active-1"),
    });
    expect(Session.Cluster.selectState(store.getState(), "active-1")).toBeDefined();
    expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual("active-1");
  });

  it("should not change the key when it already matches the server", async () => {
    mocks.connectionState = { status: "connected", clusterKey: "active-1" };
    const { store } = await renderHookWithConsole(() => Cluster.useSyncClusterKey(), {
      preloadedState: clusterState([cluster("active-1")], "active-1"),
    });
    expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual("active-1");
  });

  it("should adopt the server's cluster key once connected", async () => {
    mocks.connectionState = { status: "connected", clusterKey: "server-key" };
    const { store } = await renderHookWithConsole(() => Cluster.useSyncClusterKey(), {
      preloadedState: clusterState([cluster("active-1")], "active-1"),
    });
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toEqual("server-key"),
    );
    expect(Session.Cluster.selectState(store.getState(), "active-1")).toBeUndefined();
  });
});
