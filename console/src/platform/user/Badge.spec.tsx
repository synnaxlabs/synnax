// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/platform/user";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const clusterState = (username: string, selected = "LOCAL") => ({
  [Session.Cluster.SLICE_NAME]: {
    version: 0 as const,
    selected,
    clusters: {
      LOCAL: {
        key: "LOCAL",
        name: "Local",
        host: "localhost",
        port: 9090,
        username,
        password: "seldon",
        secure: false,
      },
    },
  },
});

describe("User.Badge", () => {
  it("should fall back to the cluster username when no user is loaded", async () => {
    await renderWithConsole(<User.Badge />, {
      preloadedState: clusterState("cluster-user"),
    });
    expect(screen.getByText("cluster-user")).toBeTruthy();
  });

  it("should render a trigger even when there is no user or cluster", async () => {
    const { container } = await renderWithConsole(<User.Badge />);
    expect(container.querySelector("button")).not.toBeNull();
  });

  it("should show the retrieved user's identity against a live cluster", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: clusterState("synnax"),
    });
    render(<User.Badge />, { wrapper });
    await waitFor(() => expect(screen.getByText("synnax")).toBeTruthy(), TIMEOUT);
  });

  it("should log out of the active cluster when Log out is clicked", async () => {
    const { store } = await renderWithConsole(<User.Badge />, {
      preloadedState: clusterState("cluster-user"),
    });
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("LOCAL");
    fireEvent.click(screen.getByText("cluster-user"));
    const logout = await screen.findByText("Log out");
    fireEvent.click(logout);
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
