// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { findCommand } from "@/platform/command/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { assertDefined, renderHookWithConsole, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("LinePlot Commands", () => {
  it("creates a line plot on the server and opens it as a tab", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: LinePlot.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create line plot");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource")
      throw new Error("focused tab is not a line plot resource");
    expect(tab.resource.type).toBe(lineplot.TYPE_ONTOLOGY_ID.type);
    const created = await client.lineplots.retrieve(tab.resource.key);
    expect(created.name).toBe("Line plot");
  });
});

describe("LinePlot Commands permissions", () => {
  it("should offer Create a line plot to an engineer", async () => {
    const gate = findCommand(LinePlot.COMMANDS, "Create line plot").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Create a line plot from a viewer", async () => {
    const gate = findCommand(LinePlot.COMMANDS, "Create line plot").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(lineplot.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
