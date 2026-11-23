// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { canvas } from "@synnaxlabs/pluto/ether";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Vis } from "@/platform/vis";
import { createConsoleWrapper, installMissingGPU } from "@/testutil";

const renderBoundary = async (inCanvas: boolean) => {
  const { wrapper } = await createConsoleWrapper({
    client: null,
    additionalRegistry: canvas.REGISTRY,
  });
  const boundary = (
    <Vis.GPUBoundary name="Plots">
      <span>Plot</span>
    </Vis.GPUBoundary>
  );
  return render(inCanvas ? <Vis.Canvas>{boundary}</Vis.Canvas> : boundary, { wrapper });
};

describe("Vis.GPUBoundary", () => {
  it("should render its children when nothing reports a missing GPU", async () => {
    await renderBoundary(false);
    expect(screen.getByText("Plot")).toBeDefined();
    expect(screen.queryByText("GPU required")).toBeNull();
  });

  describe("when the canvas has no GPU", () => {
    installMissingGPU();

    it("should show the status with the name in place of its children", async () => {
      await renderBoundary(true);
      expect(await screen.findByText("GPU required")).toBeDefined();
      expect(
        screen.getByText(
          "Plots need a GPU to render. Run the Console on a machine with a GPU.",
        ),
      ).toBeDefined();
      expect(screen.queryByText("Plot")).toBeNull();
    });
  });
});
