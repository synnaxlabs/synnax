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
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const createClusterState = (username: string, selected = "LOCAL") => ({
  [Session.Node.SLICE_NAME]: {
    version: 0 as const,
    selected,
    nodes: {
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
      preloadedState: createClusterState("cluster-user"),
    });
    expect(screen.getByText("cluster-user")).toBeTruthy();
  });

  it("should prefer the retrieved user's identity over the cluster fallback", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: createClusterState("fallback_user"),
    });
    render(<User.Badge />, { wrapper });
    await waitFor(() => expect(screen.getByText("synnax")).toBeTruthy());
    expect(screen.queryByText("fallback_user")).toBeNull();
  });

  it("should log out of the active cluster when Log out is clicked", async () => {
    const { store } = await renderWithConsole(<User.Badge />, {
      preloadedState: createClusterState("cluster-user"),
    });
    expect(Session.Node.selectSelectedKey(store.getState())).toBe("LOCAL");
    fireEvent.click(screen.getByText("cluster-user"));
    const logout = await screen.findByText("Log out");
    fireEvent.click(logout);
    await waitFor(() =>
      expect(Session.Node.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
