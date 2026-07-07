// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { stubGeometry, waitForPlacedLayout } from "@/testutil";

stubGeometry();

describe("LinePlot Commands", () => {
  it("creates a line plot on the server and places its layout", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: LinePlot.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create a line plot");
    const key = await waitForPlacedLayout(store, LinePlot.LAYOUT_TYPE);
    const created = await client.lineplots.retrieve({ key });
    expect(created.name).toBe("Line Plot");
  });
});
