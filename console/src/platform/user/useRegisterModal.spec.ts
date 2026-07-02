// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/platform/user";
import { openModal } from "@/platform/user/testutil";

const TIMEOUT = { timeout: 5000 };

const registerButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByText("Register")
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error("Register button not found");
  return btn;
};

describe("User.useRegisterModal", () => {
  it("should render every registration field", async () => {
    await openModal(User.useRegisterModal);
    await screen.findByText("Username");
    expect(screen.getByText("First")).toBeTruthy();
    expect(screen.getByText("Last")).toBeTruthy();
    expect(screen.getByText("Password")).toBeTruthy();
    expect(screen.getByText("Role")).toBeTruthy();
  });

  it("should disable the Register button when no cluster is connected", async () => {
    await openModal(User.useRegisterModal);
    await screen.findByText("Username");
    expect(registerButton().className).toContain("pluto--disabled");
  });

  it("should enable the Register button against a live cluster", async () => {
    const client = createTestClient();
    await openModal(User.useRegisterModal, { client });
    await waitFor(
      () => expect(registerButton().className).not.toContain("pluto--disabled"),
      TIMEOUT,
    );
  });
});
