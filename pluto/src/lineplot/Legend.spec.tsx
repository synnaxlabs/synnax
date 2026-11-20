// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot as client } from "@synnaxlabs/client";
import { color, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { render } from "@/testutil";

interface LegendLineProps {
  lineKey: string;
  legendGroup: string;
  label?: string;
  axis?: client.AxisKey;
}

const LegendLine = ({
  lineKey,
  legendGroup,
  label = lineKey,
  axis,
}: LegendLineProps): null => {
  const { setLine, removeLine } = LinePlot.useContext("Line");
  useEffect(() => {
    setLine({
      key: lineKey,
      color: color.construct("#ff0000"),
      label,
      visible: true,
      legendGroup,
      axis,
    });
    return () => removeLine(lineKey);
  }, []);
  return null;
};

const renderLegend = (
  lines: LegendLineProps[],
  props: LinePlot.LegendProps = {},
): HTMLElement =>
  render(
    <LinePlot.Frame aetherKey="plot">
      {lines.map((l): ReactElement => (
        <LegendLine key={l.lineKey} {...l} />
      ))}
      <LinePlot.Legend {...props} />
    </LinePlot.Frame>,
    { render: true, registry: lineplot.REGISTRY },
  ).container;

const ZERO = TimeStamp.seconds(100);

describe("Legend", () => {
  describe("absolute", () => {
    it("should name each group by its axis", async () => {
      const container = renderLegend([
        { lineKey: "a", legendGroup: "Y2" },
        { lineKey: "b", legendGroup: "Y3" },
      ]);
      await waitFor(() => expect(container.textContent).toContain("Y3"));
      expect(container.textContent).not.toContain("Y1");
    });

    it("should list the lines of every group", async () => {
      const container = renderLegend([
        { lineKey: "a", legendGroup: "Y1" },
        { lineKey: "b", legendGroup: "Y2" },
        { lineKey: "c", legendGroup: "Y3" },
      ]);
      await waitFor(() => expect(container.textContent).toContain("b"));
      expect(container.textContent).toContain("c");
    });

    it("should report the hover of a line and not of an axis name", async () => {
      const onHoverChange = vi.fn();
      renderLegend(
        [
          { lineKey: "a", legendGroup: "Y1" },
          { lineKey: "b", legendGroup: "Y2" },
        ],
        { onHoverChange },
      );
      fireEvent.pointerEnter(await screen.findByText("a"));
      expect(onHoverChange).toHaveBeenLastCalledWith("a");
      fireEvent.pointerEnter(screen.getByText("Y2"));
      expect(onHoverChange).toHaveBeenCalledTimes(1);
    });
  });

  describe("relative", () => {
    const LINES: LegendLineProps[] = [
      { lineKey: "a/r1", legendGroup: "r1", label: "pressure", axis: "y1" },
      { lineKey: "a/r2", legendGroup: "r2", label: "pressure", axis: "y1" },
    ];
    const RANGES: LinePlot.LegendRange[] = [
      { key: "r2", name: "Test 2", color: "#00ff00", zero: ZERO },
      { key: "r1", name: "Test 1", color: "#ff0000", zero: ZERO },
    ];

    it("should list the lines of each range under its name, in range order", async () => {
      const container = renderLegend(LINES, { ranges: RANGES });
      await waitFor(() => expect(container.textContent).toContain("pressure"));
      const text = container.textContent ?? "";
      expect(text.indexOf("Test 2")).toBeLessThan(text.indexOf("Test 1"));
      expect(text).toMatch(/Test 2Y1pressure.*Test 1Y1pressure/);
    });

    it("should mark a range with a custom T0", async () => {
      const ranges = [{ ...RANGES[0], start: ZERO.add(TimeStamp.seconds(5)) }];
      renderLegend(LINES, { ranges });
      expect(await screen.findByText("T0")).toBeDefined();
    });

    it("should note a range that has no zero", async () => {
      const ranges = [{ ...RANGES[0], zero: undefined }];
      renderLegend(LINES, { ranges });
      expect(await screen.findByText("Shows in absolute time only")).toBeDefined();
    });

    it("should hide a range from the eye on its name, alone on an Alt click", async () => {
      const onRangeVisibleChange = vi.fn();
      renderLegend(LINES, { ranges: RANGES, onRangeVisibleChange });
      await screen.findAllByText("pressure");
      const name = screen.getByText("Test 1").closest(".pluto-legend-name");
      const eye = name?.querySelector("button[aria-label='Hide']");
      if (eye == null) throw new Error("the range has no eye");
      fireEvent.click(eye);
      expect(onRangeVisibleChange).toHaveBeenLastCalledWith("r1", false, false);
      fireEvent.click(eye, { altKey: true });
      expect(onRangeVisibleChange).toHaveBeenLastCalledWith("r1", false, true);
    });

    it("should report the hover of a range name with the range key", async () => {
      const onHoverChange = vi.fn();
      renderLegend(LINES, { ranges: RANGES, onHoverChange });
      const name = await screen.findByText("Test 2");
      fireEvent.pointerEnter(name);
      expect(onHoverChange).toHaveBeenLastCalledWith("r2");
      fireEvent.pointerLeave(name);
      expect(onHoverChange).toHaveBeenLastCalledWith(null);
    });
  });
});
