// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { render } from "@/testutil";

interface LegendLineProps {
  lineKey: string;
  legendGroup: string;
}

const LegendLine = ({ lineKey, legendGroup }: LegendLineProps): null => {
  const { setLine, removeLine } = LinePlot.useContext("Line");
  useEffect(() => {
    setLine({
      key: lineKey,
      color: color.construct("#ff0000"),
      label: lineKey,
      visible: true,
      legendGroup,
    });
    return () => removeLine(lineKey);
  }, []);
  return null;
};

const renderLegend = (lines: LegendLineProps[]): HTMLElement =>
  render(
    <LinePlot.Frame aetherKey="plot">
      {lines.map((l): ReactElement => (
        <LegendLine key={l.lineKey} {...l} />
      ))}
      <LinePlot.Legend variant="fixed" />
    </LinePlot.Frame>,
    { render: true, registry: lineplot.REGISTRY },
  ).container;

describe("Legend", () => {
  describe("fixed", () => {
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
  });
});
