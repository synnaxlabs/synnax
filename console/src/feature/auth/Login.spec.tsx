// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { assert, describe, expect, it, vi } from "vitest";

// A Core serving the Console is its own login target, so the step is skipped
// entirely. Detection has no seam other than the module itself, and it reports a
// serving Core for every browser build, this suite included.
vi.mock("@/platform/cluster/detectConnection", () => ({
  detectConnection: () => null,
}));

import { Login } from "@/feature/auth/Login";
import { Session } from "@/session";
import { createCluster } from "@/session/cluster/testutil";
import { createSessionConsoleWrapper, getBySelector } from "@/testutil";

// Both Cores are unreachable. The list checks each one, but this suite is about
// moving between the steps, not about what the checks report.
const ALPHA = createCluster("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCluster("beta", { name: "Beta", port: 9099, username: "grace" });

const BACK = ".console-login__back";
const USERNAME_PLACEHOLDER = "synnax";

const renderLogin = async (): Promise<HTMLElement> => {
  const { wrapper } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: {
      [Session.Cluster.SLICE_NAME]: {
        ...Session.Cluster.ZERO_SLICE_STATE,
        clusters: { [ALPHA.key]: ALPHA, [BETA.key]: BETA },
        selected: undefined,
      },
    },
  });
  const { container } = render(<Login />, { wrapper });
  return container;
};

describe("auth/Login", () => {
  it("should start on the Core list when no Core is selected", async () => {
    await renderLogin();
    expect(await screen.findByText(ALPHA.name)).toBeTruthy();
    expect(screen.getByText(BETA.name)).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should advance to the credentials with the Core's saved username", async () => {
    await renderLogin();
    fireEvent.click(await screen.findByText(BETA.name));
    const username = await screen.findByPlaceholderText(USERNAME_PLACEHOLDER);
    assert(username instanceof HTMLInputElement);
    expect(username.value).toBe(BETA.username);
    expect(screen.queryByText(ALPHA.name)).toBeNull();
  });

  it("should return to the Core list from the credentials", async () => {
    const container = await renderLogin();
    fireEvent.click(await screen.findByText(ALPHA.name));
    await screen.findByPlaceholderText(USERNAME_PLACEHOLDER);
    fireEvent.click(getBySelector(container, BACK));
    expect(await screen.findByText(BETA.name)).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });
});
