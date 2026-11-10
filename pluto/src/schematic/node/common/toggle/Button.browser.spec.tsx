// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { height, renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";

describe("Toggle.Button", () => {
  it("should leave no gap around its drawing, so handles stay on it", async () => {
    const { container } = await renderThemed(
      <div data-testid="node">
        <Toggle.Button>
          <Primitive.SVG dimensions={{ width: 40, height: 40 }} />
        </Toggle.Button>
      </div>,
    );
    const node = container.querySelector('[data-testid="node"]');
    const svg = container.querySelector("svg");
    if (node == null || svg == null) throw new Error("toggle not rendered");
    expect(height(node)).toBeCloseTo(height(svg));
  });
});
