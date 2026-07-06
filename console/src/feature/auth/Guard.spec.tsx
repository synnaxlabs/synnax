// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Auth } from "@/feature/auth";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  pinLocationOrigin,
  renderWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

const CLUSTER_KEY = "local";

const clusterState = (): Session.Node.SliceState => ({
  ...Session.Node.ZERO_SLICE_STATE,
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
        [Session.Node.SLICE_NAME]: {
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
      const key = Session.Node.selectSelectedKey(store.getState());
      expect(key).toBeDefined();
      expect(key).not.toBe(CLUSTER_KEY);
    });
    expect(await screen.findByText("authenticated content")).toBeTruthy();
    const state = store.getState();
    const key = Session.Node.selectSelectedKey(state);
    if (key == null) throw new Error("no cluster selected");
    const cluster = Session.Node.selectState(state, key);
    expect(cluster?.username).toBe("synnax");
  });

  it("should surface an error status when credentials are rejected", async () => {
    pinLocationOrigin("http://localhost:9090");
    const store = await renderGuard();
    submitCredentials("synnax", uniqueName("wrong"));
    expect(await screen.findByText(/invalid credentials/i)).toBeTruthy();
    expect(Session.Node.selectSelectedKey(store.getState())).toBeUndefined();
  });
});
