// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Schematic } from "@/feature/schematic";
import { client, testProjectKey } from "@/feature/schematic/testutil";
import { findCommand } from "@/platform/command/testutil";
import { Session } from "@/session";
import { assertDefined, renderHookWithConsole, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("Schematic Commands", () => {
  it("is visible when the user may create schematics", async () => {
    const { result } = await renderHookWithConsole(
      () => Schematic.COMMANDS[0].useVisible?.(),
      { client },
    );
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("creates and places a schematic when selected", async () => {
    const project = await testProjectKey();
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Schematic.COMMANDS,
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: project,
        },
      },
    });
    await openCommandPalette();
    await selectCommand("Create a schematic");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource")
      throw new Error("focused tab is not a schematic resource");
    const created = await client.schematics.retrieve(tab.resource.key);
    expect(created.name).toBe("Schematic");
  });
});

describe("Schematic Commands permissions", () => {
  it("should offer Create a schematic to an engineer", async () => {
    const gate = findCommand(Schematic.COMMANDS, "Create a schematic").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Create a schematic from a viewer", async () => {
    const gate = findCommand(Schematic.COMMANDS, "Create a schematic").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(schematic.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
