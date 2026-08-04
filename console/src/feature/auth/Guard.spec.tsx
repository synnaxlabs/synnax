// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Auth } from "@/feature/auth";
import { Cluster } from "@/feature/cluster";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  createSessionConsoleWrapper,
  pinLocationOrigin,
  renderWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

const CLUSTER_KEY = "local";

const clusterState = (): Session.Cluster.SliceState => ({
  ...Session.Cluster.ZERO_SLICE_STATE,
  clusters: {
    [CLUSTER_KEY]: {
      key: CLUSTER_KEY,
      name: "Local",
      host: "localhost",
      port: 9090,
      username: "",
      password: "",
      secure: false,
    },
  },
});

const renderGuard = async (selected?: string): Promise<TestStore> => {
  const { store } = await renderWithConsole(
    <Auth.Guard>
      <span>authenticated content</span>
    </Auth.Guard>,
    {
      preloadedState: {
        [Session.Cluster.SLICE_NAME]: {
          ...clusterState(),
          selected,
        },
      },
    },
  );
  return store;
};

const submitCredentials = (username: string, password: string): void => {
  fireEvent.change(screen.getByPlaceholderText("synnax"), {
    target: { value: username },
  });
  fireEvent.change(screen.getByPlaceholderText("seldon"), {
    target: { value: password },
  });
  fireEvent.click(findButton("Log In"));
};

describe("auth guard", () => {
  it("should render children when a cluster is already selected", async () => {
    await renderGuard(CLUSTER_KEY);
    expect(screen.getByText("authenticated content")).toBeTruthy();
    expect(screen.queryByText("Log In")).toBeNull();
  });

  it("should render the login screen when no cluster is selected", async () => {
    await renderGuard();
    expect(screen.getAllByText("Log In").length).toBeGreaterThan(0);
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it("should select the cluster after a successful login", async () => {
    pinLocationOrigin("http://localhost:9090");
    const store = await renderGuard();
    submitCredentials("synnax", "seldon");
    await waitFor(() => {
      const key = Session.Cluster.selectSelectedKey(store.getState());
      expect(key).toBeDefined();
      expect(key).not.toBe(CLUSTER_KEY);
    });
    expect(await screen.findByText("authenticated content")).toBeTruthy();
    const state = store.getState();
    const key = Session.Cluster.selectSelectedKey(state);
    if (key == null) throw new Error("no cluster selected");
    const cluster = Session.Cluster.selectState(state, key);
    expect(cluster?.username).toBe("synnax");
  });

  it("should return to the login surface when credentials are rejected", async () => {
    pinLocationOrigin("http://localhost:9090");
    const { wrapper, store } = await createSessionConsoleWrapper({ client: null });
    render(
      <Session.Settled.Provider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <Cluster.ConnectionBadge />
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.Settled.Provider>,
      { wrapper },
    );
    submitCredentials("synnax", uniqueName("wrong"));
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toBeDefined(),
    );
    const key = Session.Cluster.selectSelectedKey(store.getState());
    expect(await screen.findByText(/invalid credentials/i)).toBeTruthy();
    expect(screen.getAllByText("Log In").length).toBeGreaterThan(0);
    expect(screen.queryByText("authenticated content")).toBeNull();
    submitCredentials("synnax", "seldon");
    await waitFor(
      () => expect(document.querySelector(".pluto--status-success")).toBeTruthy(),
      { timeout: 10000 },
    );
    // The login sequence swaps the cluster and project partitions in turn;
    // the workspace settles once the second swap hydrates.
    expect(
      await screen.findByText("authenticated content", {}, { timeout: 10000 }),
    ).toBeTruthy();
    expect(screen.queryByText(/invalid credentials/i)).toBeNull();
    // The synchronizer adopts the server-issued key, renaming the failed
    // login's entry in place rather than minting a duplicate.
    const selected = Session.Cluster.selectSelectedKey(store.getState());
    expect(selected).toBeDefined();
    expect(selected).not.toBe(key);
    const keys = Session.Cluster.selectMany(store.getState())
      .map(({ key: k }) => k)
      .filter((k) => k !== "DEMO");
    expect(keys).toEqual([selected]);
  });

  it("should surface connection trouble at a cold start against an unreachable cluster", async () => {
    const DEAD_KEY = "dead";
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
          clusters: {
            [DEAD_KEY]: {
              key: DEAD_KEY,
              name: "Dead",
              host: "localhost",
              port: 9098,
              username: "synnax",
              password: "seldon",
              secure: false,
            },
          },
          selected: DEAD_KEY,
        },
      },
    });
    render(
      <Session.Settled.Provider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.Settled.Provider>,
      { wrapper },
    );
    expect(await screen.findByText("Retry Now", {}, { timeout: 10000 })).toBeTruthy();
    expect(screen.getAllByText("localhost:9098").length).toBeGreaterThan(0);
    expect(screen.queryByText("Preparing your workspace...")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
  });
});
