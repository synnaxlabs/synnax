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

// A serving Core skips the Core step, and detection reports one for every browser
// build. The module is the only seam.
vi.mock("@/platform/core/detectConnection", () => ({
  detectConnection: () => null,
}));

import { Login } from "@/feature/auth/Login";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import { createSessionConsoleWrapper, getBySelector } from "@/testutil";

// Dead ports: the list checks each Core, and no assertion here reads the result.
const ALPHA = createCore("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCore("beta", { name: "Beta", port: 9099, username: "grace" });

const BACK = ".console-login__back";
const USERNAME_PLACEHOLDER = "synnax";

const renderLogin = async (): Promise<HTMLElement> => {
  const { wrapper } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: {
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        cores: { [ALPHA.key]: ALPHA, [BETA.key]: BETA },
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
