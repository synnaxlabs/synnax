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

import { Table } from "@/feature/table/Table";
import { renderTable } from "@/feature/table/testutil";
import { Vis } from "@/platform/vis";
import { installMissingGPU } from "@/testutil";

describe("table/Table", () => {
  describe("when the canvas has no GPU", () => {
    installMissingGPU();

    it("shows a placeholder in place of the table", async () => {
      const InCanvas = () => (
        <Vis.Canvas>
          <Table />
        </Vis.Canvas>
      );
      await renderTable(InCanvas, { additionalRegistry: canvas.REGISTRY });
      expect(await screen.findByText("GPU required")).toBeDefined();
      expect(
        screen.getByText(
          "Tables need a GPU to render. Run the Console on a machine with a GPU.",
        ),
      ).toBeDefined();
    });
  });
});
