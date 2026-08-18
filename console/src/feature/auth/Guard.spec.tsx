// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Auth } from "@/feature/auth";
import { Cluster } from "@/feature/cluster";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCluster, createClusterState } from "@/session/cluster/testutil";
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
  fireEvent.click(findButton("Log in"));
};

describe("auth guard", () => {
  it("should render children when a cluster is already selected", async () => {
    await renderGuard(CLUSTER_KEY);
    expect(screen.getByText("authenticated content")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
  });

  it("should render the login screen when no cluster is selected", async () => {
    await renderGuard();
    expect(screen.getAllByText("Log in").length).toBeGreaterThan(0);
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
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <Cluster.ConnectionBadge />
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    submitCredentials("synnax", uniqueName("wrong"));
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toBeDefined(),
    );
    const key = Session.Cluster.selectSelectedKey(store.getState());
    expect(await screen.findByText(/invalid credentials/i)).toBeTruthy();
    expect(screen.getAllByText("Log in").length).toBeGreaterThan(0);
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
            [DEAD_KEY]: createCluster(DEAD_KEY, { name: "Dead", port: 9098 }),
          },
          selected: DEAD_KEY,
        },
      },
    });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    expect(await screen.findByText("Retry now", {}, { timeout: 10000 })).toBeTruthy();
    expect(screen.getAllByText("localhost:9098").length).toBeGreaterThan(0);
    expect(screen.queryByText("Preparing your workspace...")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it("should disable the retry button while a check is in flight", async () => {
    const DEAD_KEY = "dead";
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
          clusters: {
            [DEAD_KEY]: createCluster(DEAD_KEY, { name: "Dead", port: 9098 }),
          },
          selected: DEAD_KEY,
        },
      },
    });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    await screen.findByText("Retry now", {}, { timeout: 10000 });
    // The scheduled retries raise the same flag, so the assertion starts from a
    // gap between them rather than from whatever the countdown happens to be doing.
    await waitFor(
      () => expect(findButton("Retry now").getAttribute("aria-disabled")).toBeNull(),
      { timeout: 10000 },
    );
    fireEvent.click(findButton("Retry now"));
    await waitFor(() =>
      expect(findButton("Retry now").getAttribute("aria-disabled")).toBe("true"),
    );
  });
});

describe("connection guard permissions", () => {
  const createGuardWrapper = async (): Promise<FC<PropsWithChildren>> => {
    const { wrapper: SessionWrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createClusterState([createCluster(CLUSTER_KEY)], CLUSTER_KEY),
    });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <SessionWrapper>
        <Session.SettledProvider>
          <Auth.Guard>
            <Auth.ConnectionGuard>{children}</Auth.ConnectionGuard>
          </Auth.Guard>
        </Session.SettledProvider>
      </SessionWrapper>
    );
    Wrapper.displayName = "GuardWrapper";
    return Wrapper;
  };

  it("should never render a guarded child against an empty policy set", async () => {
    const wrapper = await createGuardWrapper();
    const verdicts: boolean[] = [];
    const { result } = renderHook(
      () => {
        const client = Synnax.use();
        const granted = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
        if (client != null) verdicts.push(granted);
        return granted;
      },
      { wrapper },
    );
    await waitFor(() => expect(result.current).toBe(true), { timeout: 10000 });
    expect(verdicts).not.toContain(false);
  });

  it("should hold the splash while the policies load", async () => {
    const wrapper = await createGuardWrapper();
    render(<span>authenticated content</span>, { wrapper });
    expect(screen.queryByText("authenticated content")).toBeNull();
    expect(
      await screen.findByText("authenticated content", {}, { timeout: 10000 }),
    ).toBeTruthy();
  });
});
