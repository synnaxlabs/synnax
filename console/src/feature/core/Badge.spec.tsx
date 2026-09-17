// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/feature/core";
import { clickCoreBadge, hoverCoreBadge } from "@/feature/core/testutil";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import {
  createConnectedConsoleWrapper,
  createConsoleWrapper,
  renderWithConsole,
} from "@/testutil";

const CORE_KEY = "core-key";

const createStateWithUser = (username: string) =>
  createCoreState([createCore("Local", { key: CORE_KEY, username })], CORE_KEY);

describe("Core.Badge", () => {
  it("should report disconnected when no Core connection exists", async () => {
    const { container } = await renderWithConsole(<Core.Badge />);
    hoverCoreBadge(container);
    expect((await screen.findAllByText("Disconnected")).length).toBeGreaterThan(0);
  });

  it("should report connected once the provider reaches the Core", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    const { container } = render(<Core.Badge />, { wrapper });
    hoverCoreBadge(container);
    expect(await screen.findByText("Connected")).toBeTruthy();
  });

  it("should show the signed-in user's name in the trigger", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    render(<Core.Badge />, { wrapper });
    expect(await screen.findByText("synnax")).toBeTruthy();
  });

  it("should fall back to the Core username in the trigger when no user is loaded", async () => {
    await renderWithConsole(<Core.Badge />, {
      preloadedState: createStateWithUser("Core-user"),
    });
    expect(screen.getByText("Core-user")).toBeTruthy();
  });

  it("should open the diagnostics dialog on click", async () => {
    const { container } = await renderWithConsole(<Core.Badge />);
    clickCoreBadge(container);
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getAllByText("Disconnected").length).toBeGreaterThan(0);
  });

  it("should show server facts in the dialog once connected", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    const { container } = render(<Core.Badge />, { wrapper });
    clickCoreBadge(container);
    const dialog = await screen.findByRole("dialog");
    await within(dialog).findByText("Connected");
    expect(within(dialog).getByText(/^Core v/)).toBeTruthy();
  });

  it("should close the dialog and open the connect modal from the name row", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: null,
      preloadedState: createStateWithUser("Core-user"),
    });
    const { container } = render(
      <Triggers.Provider>
        <Core.Badge />
        <Modals.Stack />
      </Triggers.Provider>,
      { wrapper },
    );
    clickCoreBadge(container);
    fireEvent.click(await screen.findByLabelText("Edit connection"));
    // "Save" is the edit-mode submit, so the modal opened on the active Core rather
    // than in create mode.
    expect(await screen.findByRole("button", { name: "Save" })).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
  });

  it("should log out of the active Core when Log out is clicked", async () => {
    const { container, store } = await renderWithConsole(<Core.Badge />, {
      preloadedState: createStateWithUser("Core-user"),
    });
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(CORE_KEY);
    clickCoreBadge(container);
    fireEvent.click(await screen.findByText("Log out"));
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
