// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { canvas } from "@synnaxlabs/pluto/ether";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic/Schematic";
import { renderSchematic } from "@/feature/schematic/testutil";
import { Vis } from "@/platform/vis";
import { installMissingGPU } from "@/testutil";

describe("schematic/Schematic", () => {
  describe("when the canvas has no GPU", () => {
    installMissingGPU();

    it("shows a placeholder in place of the schematic", async () => {
      const InCanvas = () => (
        <Vis.Canvas>
          <Schematic />
        </Vis.Canvas>
      );
      await renderSchematic(InCanvas, { additionalRegistry: canvas.REGISTRY });
      expect(await screen.findByText("GPU required")).toBeDefined();
      expect(
        screen.getByText(
          "Schematics need a GPU to render. Run the Console on a machine with a GPU.",
        ),
      ).toBeDefined();
    });
  });
});
