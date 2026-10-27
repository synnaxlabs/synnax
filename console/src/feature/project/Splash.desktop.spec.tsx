// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Modals } from "@/platform/modals";
import { findButton, getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

describe("project/Splash", () => {
  it("should offer no log out and name no Core", async () => {
    const { store } = await renderWithConsole(<Project.Splash />);
    act(() => {
      store.dispatch(Session.Core.select("LOCAL"));
    });
    const name = Session.Core.selectSelected(store.getState())?.name;
    if (name == null) throw new Error("no Core is selected");
    expect(screen.getByText("Projects")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Log out" })).toBeNull();
    expect(screen.queryByText(name)).toBeNull();
  });

  it("should not offer the sample project when projects exist", async () => {
    await client.projects.create({ name: uniqueName("proj") });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Project.Splash />, { wrapper });
    await screen.findByPlaceholderText("Search projects...");
    expect(screen.queryByRole("button", { name: "Sample project" })).toBeNull();
    expect(screen.queryByText("Open sample project")).toBeNull();
  });

  it("should open the usage data settings on a linked machine", async () => {
    await renderWithConsole(
      <>
        <Modals.Stack />
        <Project.Splash />
      </>,
      {
        preloadedState: {
          [Session.Account.SLICE_NAME]: {
            ...Session.Account.ZERO_SLICE_STATE,
            link: { secret: "shh", email: "someone@example.com", user: "user_a" },
          },
        },
      },
    );
    fireEvent.click(findButton("Usage data"));
    await screen.findByText("Share usage data");
    expect(getSwitch("Share usage data").checked).toBe(false);
  });

  it("should not offer usage data on a machine not linked to an account", async () => {
    await renderWithConsole(<Project.Splash />);
    expect(screen.queryByText("Usage data")).toBeNull();
  });
});
