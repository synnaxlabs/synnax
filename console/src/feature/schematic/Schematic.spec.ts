// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer, panel, project, schematic } from "@synnaxlabs/client";
import { aether } from "@synnaxlabs/pluto/ether";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Schematic } from "@/feature/schematic/Schematic";
import { client, renderSchematic } from "@/feature/schematic/testutil";
import { createTestClientWithGrants, type Grants } from "@/testutil";

const stateZ = z.looseObject({});

// The real diagram worker needs a canvas, and the published ether entry does not export
// the control workers.
class DiagramStub extends aether.Composite<typeof stateZ> {
  static readonly TYPE = "Diagram";
  schema = stateZ;
}

class ControllerStub extends aether.Composite<typeof stateZ> {
  static readonly TYPE = "Controller";
  schema = stateZ;
}

class LegendStub extends aether.Leaf<typeof stateZ> {
  static readonly TYPE = "Legend";
  schema = stateZ;
}

const REGISTRY = {
  [DiagramStub.TYPE]: DiagramStub,
  [ControllerStub.TYPE]: ControllerStub,
  [LegendStub.TYPE]: LegendStub,
};

const EDIT: Grants = {
  retrieve: [
    panel.TYPE_RESOURCE_ID,
    project.TYPE_RESOURCE_ID,
    schematic.TYPE_RESOURCE_ID,
  ],
  update: [schematic.TYPE_RESOURCE_ID],
};

const openMenu = async (grants: Grants): Promise<void> => {
  await renderSchematic(Schematic, {
    as: await createTestClientWithGrants(client, grants),
    additionalRegistry: REGISTRY,
    sessionState: { editable: false },
    visible: true,
  });
  fireEvent.contextMenu(await screen.findByText("No symbols in this schematic"));
  // Both items resolve a permission query, so the edit item proves the menu settled.
  await screen.findByText("Enable editing");
};

describe("Schematic", () => {
  it("should offer control to a user who can command", async () => {
    await openMenu({ ...EDIT, create: [framer.TYPE_RESOURCE_ID] });
    expect(await screen.findByText("Take control")).toBeTruthy();
  });

  it("should not offer control to a user who cannot command", async () => {
    await openMenu(EDIT);
    expect(screen.queryByText("Take control")).toBeNull();
  });
});
