// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { resolveFocusedTab, stubGeometry } from "@/testutil";

stubGeometry();

describe("LinePlot Commands", () => {
  it("creates a line plot on the server and opens it as a tab", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: LinePlot.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create a line plot");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource")
      throw new Error("focused tab is not a line plot resource");
    expect(tab.resource.type).toBe(lineplot.TYPE_ONTOLOGY_ID.type);
    const created = await client.lineplots.retrieve({ key: tab.resource.key });
    expect(created.name).toBe("New Line Plot");
  });
});
