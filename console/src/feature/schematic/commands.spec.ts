// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Schematic } from "@/feature/schematic";
import { client, testProjectKey } from "@/feature/schematic/testutil";
import { Session } from "@/session";
import { renderHookWithConsole, stubGeometry, waitForPlacedLayout } from "@/testutil";

stubGeometry();

describe("Schematic.COMMANDS", () => {
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
    const placedKey = await waitForPlacedLayout(store, "schematic");
    await waitFor(async () => {
      const created = await client.schematics.retrieve({ key: placedKey });
      expect(created.name).toBe("Schematic");
    });
  });
});
