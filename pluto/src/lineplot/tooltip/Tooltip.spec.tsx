// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";
import { Profiler } from "react";
import { afterEach, assert, beforeEach, describe, expect, it } from "vitest";

import { lineplot } from "@/lineplot/aether";
import { Frame } from "@/lineplot/Frame";
import { tooltip } from "@/lineplot/tooltip/aether";
import { Tooltip } from "@/lineplot/tooltip/Tooltip";
import { Viewport } from "@/lineplot/Viewport";
import { render } from "@/testutil";

const TOOLTIP_PATH = [
  "alamos",
  "status",
  "synnax",
  "theming",
  "telem",
  "staleness",
  "render",
  "plot",
  "tooltip",
];

describe("Tooltip", () => {
  // The tooltip places itself relative to the canvas container.
  let canvas: HTMLElement;
  beforeEach(() => {
    canvas = document.createElement("div");
    canvas.className = "pluto-canvas-container";
    document.body.appendChild(canvas);
  });
  afterEach(() => canvas.remove());

  it("should not re-render when the cursor moves", async () => {
    let renders = 0;
    const { container, root } = render(
      <Frame aetherKey="plot">
        <Viewport>
          <Profiler id="tooltip" onRender={() => renders++}>
            <Tooltip aetherKey="tooltip" />
          </Profiler>
        </Viewport>
      </Frame>,
      { render: true, registry: lineplot.REGISTRY },
    );
    const leaf = await waitFor(() => {
      const found = root.findChildAtPath(TOOLTIP_PATH);
      assert(found instanceof tooltip.Tooltip);
      return found;
    });
    const viewport = container.querySelector(".pluto-line-plot__viewport");
    assert(viewport != null);
    const baseline = renders;
    fireEvent.mouseMove(viewport, { clientX: 10, clientY: 20 });
    await waitFor(() => expect(leaf.state.position).toEqual({ x: 10, y: 20 }));
    expect(renders).toBe(baseline);
  });
});
