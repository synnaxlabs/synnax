// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Framer } from "@/feature/framer";
import { findCommand } from "@/platform/command/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

import { assertDefined, renderHookWithConsole } from "@/testutil";

describe("Framer Commands", () => {
  it("should open the delete data modal when the command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Framer.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Delete data");
    expect(await screen.findByText("Select channels to delete")).toBeTruthy();
  });
});

describe("Framer Commands permissions", () => {
  it("should offer Delete data to an engineer", async () => {
    const gate = findCommand(Framer.COMMANDS, "Delete data").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Delete data from a viewer", async () => {
    const gate = findCommand(Framer.COMMANDS, "Delete data").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(framer.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
