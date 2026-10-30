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

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { render } from "@/testutil";

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
      <LinePlot.Frame aetherKey="plot" loadingMessage="Fetching 90d of data">
        <LinePlot.Viewport />
      </LinePlot.Frame>,
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

  it("should show the corner spinner while fetching after the first load", async () => {
    const { container, root } = render(
      <LinePlot.Frame aetherKey="plot">
        <LinePlot.Viewport />
      </LinePlot.Frame>,
      { render: true, registry: lineplot.REGISTRY },
    );
    let plot: lineplot.LinePlot | null = null;
    await waitFor(() => {
      plot = root.findChildAtPath(WORKER_PATH) as lineplot.LinePlot;
      expect(plot).toBeInstanceOf(lineplot.LinePlot);
    });
    expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull();
    act(() => {
      plot?.setState((p) => ({ ...p, fetching: true }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).not.toBeNull(),
    );
    act(() => {
      plot?.setState((p) => ({ ...p, loading: true }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull(),
    );
    act(() => {
      plot?.setState((p) => ({ ...p, loading: false, fetching: false }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull(),
    );
  });
});
