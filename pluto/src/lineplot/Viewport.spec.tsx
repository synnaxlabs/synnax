// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, screen, waitFor } from "@testing-library/react";
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

const renderPlot = async (
  loadingMessage?: string,
): Promise<{ container: HTMLElement; plot: lineplot.LinePlot }> => {
  const { container, root } = render(
    <Frame aetherKey="plot" loadingMessage={loadingMessage}>
      <Viewport />
    </Frame>,
    { render: true, registry: lineplot.REGISTRY },
  );
  const plot = await waitFor(() => {
    const found = root.findChildAtPath(WORKER_PATH);
    expect(found).toBeInstanceOf(lineplot.LinePlot);
    return found as lineplot.LinePlot;
  });
  return { container, plot };
};

describe("Viewport", () => {
  it("should overlay the spinner and message only while loading", async () => {
    const { container, plot } = await renderPlot("Fetching 90d of data");
    expect(container.querySelector(".pluto-line-plot__loading")).toBeNull();
    act(() => {
      plot.setState((p) => ({ ...p, loading: true }));
    });
    await waitFor(() => {
      expect(container.querySelector(".pluto-line-plot__loading")).not.toBeNull();
      expect(container.textContent).toContain("Fetching 90d of data");
    });
    act(() => {
      plot.setState((p) => ({ ...p, loading: false }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__loading")).toBeNull(),
    );
  });

  it("should show the corner spinner while fetching after the first load", async () => {
    const { container, plot } = await renderPlot();
    expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull();
    act(() => {
      plot.setState((p) => ({ ...p, fetching: true }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).not.toBeNull(),
    );
    act(() => {
      plot.setState((p) => ({ ...p, loading: true }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull(),
    );
    act(() => {
      plot.setState((p) => ({ ...p, loading: false, fetching: false }));
    });
    await waitFor(() =>
      expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull(),
    );
  });

  it("should mark line problems in place of the corner spinner", async () => {
    const { container, plot } = await renderPlot();
    expect(screen.queryByRole("img")).toBeNull();
    const problems = [
      {
        variant: "error" as const,
        name: "PT-03",
        message: "Failed to read channel data",
        description: "channel 999 not found",
      },
    ];
    act(() => {
      plot.setState((p) => ({ ...p, fetching: true, problems }));
    });
    await waitFor(() =>
      expect(
        screen.getByRole("img", { name: "Failed to read channel data" }),
      ).toBeTruthy(),
    );
    expect(container.querySelector(".pluto-line-plot__fetching")).toBeNull();
    act(() => {
      plot.setState((p) => ({ ...p, problems: [] }));
    });
    await waitFor(() => {
      expect(screen.queryByRole("img")).toBeNull();
      expect(container.querySelector(".pluto-line-plot__fetching")).not.toBeNull();
    });
  });
});
