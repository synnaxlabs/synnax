// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { assertDefined } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Library } from "@/app/library";
import { Settings } from "@/app/settings";
import { renderPalette } from "@/feature/command/testutil";
import { findCommand } from "@/platform/command/testutil";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { createConsoleWrapper, renderHookWithConsole, renderLazy } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const { Content } = Library.TABS.library;

describe("Library", () => {
  it("should open the page of each section", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    await renderLazy(<Content />, { wrapper });
    for (const name of ["Labels", "Colors", "Enumerations", "ICDs"]) {
      fireEvent.click(await screen.findByRole("menuitem", { name }));
      expect(await screen.findByRole("heading", { name })).toBeTruthy();
    }
  });

  it("should keep its section apart from the section of Settings", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    await renderLazy(
      <>
        <Content />
        <Settings.TABS.settings.Content />
      </>,
      { wrapper },
    );
    fireEvent.click(await screen.findByRole("menuitem", { name: "Colors" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "About" }));
    expect(await screen.findByRole("heading", { name: "Colors" })).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "About" })).toBeTruthy();
  });

  it("should show the Labels page when Edit labels runs", async () => {
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "other" }],
    });
    const { wrapper, store } = await createPanelWrapper({
      client,
      panelKey: existing.key,
    });
    await primePanel(wrapper, existing.key);
    await renderLazy(<Content />, { wrapper });
    fireEvent.click(await screen.findByRole("menuitem", { name: "Colors" }));
    expect(await screen.findByRole("heading", { name: "Colors" })).toBeTruthy();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Library.COMMANDS,
      client,
      store,
    });
    await openCommandPalette();
    await selectCommand("Edit labels");
    expect(await screen.findByRole("heading", { name: "Labels" })).toBeTruthy();
  });

  it("should offer the button and Open library to an Engineer", async () => {
    const engineer = await roles.get("Engineer");
    const { wrapper } = await createConsoleWrapper({ client: engineer });
    render(<Library.Button />, { wrapper });
    expect(await screen.findByRole("button", { name: "Library" })).toBeTruthy();
    const { openCommandPalette } = await renderPalette({
      commands: Library.COMMANDS,
      client: engineer,
    });
    await openCommandPalette();
    expect(await screen.findByText("Open library")).toBeTruthy();
    expect(await screen.findByText("Edit labels")).toBeTruthy();
  });

  it("should offer Open library and withhold Edit labels from a Viewer", async () => {
    const open = findCommand(Library.COMMANDS, "Open library").useVisible;
    const edit = findCommand(Library.COMMANDS, "Edit labels").useVisible;
    assertDefined(open);
    assertDefined(edit);
    const { result } = await renderHookWithConsole(
      () => ({
        open: open(),
        edit: edit(),
        loaded: Access.useRetrieveGranted(label.TYPE_RESOURCE_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    // A Viewer reads ICDs, so the Library shows them their ICDs section.
    await waitFor(() => expect(result.current.open).toBe(true));
    expect(result.current.loaded).toBe(true);
    expect(result.current.edit).toBe(false);
  });
});
