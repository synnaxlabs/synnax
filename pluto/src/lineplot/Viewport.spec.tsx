// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { lineplot } from "@/lineplot/aether";
import { Frame } from "@/lineplot/Frame";
import { Viewport } from "@/lineplot/Viewport";
import { render } from "@/testutil/render";

const WORKER_PATH = [
  "alamos",
  "status",
  "synnax",
  "theming",
  "telem",
  "staleness",
  "render",
  "plot",
];

describe("Viewport", () => {
  it("should overlay the spinner and message only while loading", async () => {
    const { container, root } = render(
      <Frame aetherKey="plot" loadingMessage="Fetching 90d of data">
        <Viewport />
      </Frame>,
      { render: true, registry: lineplot.REGISTRY },
    );
    let plot: lineplot.LinePlot | null = null;
    await waitFor(() => {
      plot = root.findChildAtPath(WORKER_PATH) as lineplot.LinePlot;
      expect(plot).toBeInstanceOf(lineplot.LinePlot);
    });
    expect(container.querySelector(".pluto-line-plot__loading")).toBeNull();
    act(() => {
      plot?.setState((p) => ({ ...p, loading: true }));
    });
    await waitFor(() => {
      expect(container.querySelector(".pluto-line-plot__loading")).not.toBeNull();
      expect(container.textContent).toContain("Fetching 90d of data");
    });
    act(() => {
      plot?.setState((p) => ({ ...p, loading: false }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__loading")).toBeNull(),
    );
  });
});
