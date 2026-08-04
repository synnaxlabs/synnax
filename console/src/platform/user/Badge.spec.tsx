// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/platform/user";
import { Session } from "@/session";
import { createCluster, createClusterState } from "@/session/cluster/testutil";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const createStateWithUser = (username: string, selected = "LOCAL") =>
  createClusterState([createCluster("LOCAL", { name: "Local", username })], selected);

describe("User.Badge", () => {
  it("should fall back to the cluster username when no user is loaded", async () => {
    await renderWithConsole(<User.Badge />, {
      preloadedState: createStateWithUser("cluster-user"),
    });
    expect(screen.getByText("cluster-user")).toBeTruthy();
  });

  it("should prefer the retrieved user's identity over the cluster fallback", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: createStateWithUser("fallback_user"),
    });
    render(<User.Badge />, { wrapper });
    await waitFor(() => expect(screen.getByText("synnax")).toBeTruthy());
    expect(screen.queryByText("fallback_user")).toBeNull();
  });

  it("should log out of the active cluster when Log out is clicked", async () => {
    const { store } = await renderWithConsole(<User.Badge />, {
      preloadedState: createStateWithUser("cluster-user"),
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
