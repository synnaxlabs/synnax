// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { type CSSProperties, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { ZERO_CALC_STATE } from "@/components/stream/calcTimeline";
import { Diagram } from "@/components/stream/diagrams/Diagram";
import type { DiagramDef, NodeIcon } from "@/components/stream/diagrams/types";

const PRIMARY = "var(--pluto-primary-p1)";
const ERROR = "var(--pluto-error-z)";
const DIM = "var(--pluto-gray-l5)";

const icon = (id: string): NodeIcon => {
  const Icon = ({ style }: { style?: CSSProperties }): ReactElement => (
    <svg data-testid={id} style={style} />
  );
  return Icon;
};

const node = (id: string, x: number): DiagramDef["nodes"][number] => ({
  id,
  x,
  y: 100,
  w: 140,
  h: 60,
  label: id,
  icon: icon(id),
});

const DEF: DiagramDef = {
  viewBox: "0 0 900 200",
  nodes: [node("src", 100), node("mid", 400), node("out", 700)],
  edges: [
    { from: "src", to: "mid" },
    { from: "mid", to: "out" },
  ],
};

const traces = (container: HTMLElement): SVGPathElement[] => [
  ...container.querySelectorAll<SVGPathElement>("path[id]"),
];

const particles = (container: HTMLElement, trace: SVGPathElement): number =>
  container.querySelectorAll(`mpath[href="#${trace.id}"]`).length;

const crossed = (trace: SVGPathElement): boolean =>
  trace.parentElement!.querySelectorAll(`line[stroke="${ERROR}"]`).length === 2;

describe("Diagram", () => {
  describe("traces", () => {
    it("should draw one trace per edge from tab to tab", () => {
      const { container } = render(<Diagram def={DEF} state={ZERO_CALC_STATE} />);
      expect(traces(container).map((t) => t.getAttribute("d"))).toEqual([
        "M180,100 L320,100",
        "M480,100 L620,100",
      ]);
    });

    it("should flow particles only along edges that touch an active node", () => {
      const { container } = render(
        <Diagram def={DEF} state={{ ...ZERO_CALC_STATE, activeNodes: ["src"] }} />,
      );
      const [first, second] = traces(container);
      expect(particles(container, first)).toBe(4);
      expect(particles(container, second)).toBe(0);
    });

    it("should flow particles into an active node too", () => {
      const { container } = render(
        <Diagram def={DEF} state={{ ...ZERO_CALC_STATE, activeNodes: ["out"] }} />,
      );
      const [first, second] = traces(container);
      expect(particles(container, first)).toBe(0);
      expect(particles(container, second)).toBe(4);
    });

    it("should cross out the edges of an excluded node and stop their flow", () => {
      const { container } = render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, activeNodes: ["src"], excludedNodes: ["src"] }}
        />,
      );
      const [first, second] = traces(container);
      expect(particles(container, first)).toBe(0);
      expect(crossed(first)).toBe(true);
      expect(crossed(second)).toBe(false);
    });

    it("should give each diagram its own trace IDs", () => {
      const { container } = render(
        <>
          <Diagram def={DEF} state={{ ...ZERO_CALC_STATE, activeNodes: ["mid"] }} />
          <Diagram def={DEF} state={{ ...ZERO_CALC_STATE, activeNodes: ["mid"] }} />
        </>,
      );
      const ids = traces(container).map((t) => t.id);
      expect(new Set(ids).size).toBe(4);
      for (const svg of container.querySelectorAll("svg[viewBox]"))
        for (const mpath of svg.querySelectorAll("mpath"))
          expect(svg.querySelector(mpath.getAttribute("href")!)).not.toBeNull();
    });
  });

  describe("tabs", () => {
    it("should give each node a tab on each side that has an edge", () => {
      const { container } = render(<Diagram def={DEF} state={ZERO_CALC_STATE} />);
      const xs = [...container.querySelectorAll('rect[height="18"]')].map((r) =>
        Number(r.getAttribute("x")),
      );
      // Right tabs overlap the node by 14; left tabs extend 10 past it.
      expect(xs.sort((a, b) => a - b)).toEqual([156, 320, 456, 620]);
    });
  });

  describe("nodes", () => {
    it("should show the value of each node that has one", () => {
      render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, nodeValues: { mid: "42 psi" } }}
        />,
      );
      expect(screen.getByText("42 psi")).toBeTruthy();
    });

    it("should color each node by its state", () => {
      render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, activeNodes: ["src"], alarmNodes: ["out"] }}
        />,
      );
      expect(screen.getByTestId("src").style.color).toBe(PRIMARY);
      expect(screen.getByTestId("mid").style.color).toBe(DIM);
      expect(screen.getByTestId("out").style.color).toBe(ERROR);
    });

    it("should color an excluded node in the error color", () => {
      render(
        <Diagram def={DEF} state={{ ...ZERO_CALC_STATE, excludedNodes: ["mid"] }} />,
      );
      expect(screen.getByTestId("mid").style.color).toBe(ERROR);
    });
  });

  describe("pill variant", () => {
    const PILL: DiagramDef = { ...DEF, variant: "pill" };

    it("should end its traces at the smaller pill tabs", () => {
      const { container } = render(<Diagram def={PILL} state={ZERO_CALC_STATE} />);
      expect(traces(container)[0].getAttribute("d")).toBe("M176,100 L324,100");
    });

    it("should draw an active pill in white with a glow and no label", () => {
      const { container } = render(
        <Diagram
          def={PILL}
          state={{
            ...ZERO_CALC_STATE,
            activeNodes: ["mid"],
            nodeValues: { mid: "hub" },
          }}
        />,
      );
      const glow = container.querySelector("filter")!.id;
      const pill = container.querySelector(`rect[filter="url(#${glow})"]`)!;
      expect(pill.getAttribute("fill")).toBe("white");
      expect(screen.queryByText("mid")).toBeNull();
      expect(screen.queryByText("hub")).toBeNull();
    });

    it("should show the label and description of an inactive pill", () => {
      render(
        <Diagram
          def={PILL}
          state={{
            ...ZERO_CALC_STATE,
            activeNodes: ["mid"],
            nodeValues: { src: "NI" },
          }}
        />,
      );
      expect(screen.getByText("src")).toBeTruthy();
      expect(screen.getByText("NI")).toBeTruthy();
    });

    it("should color every pill tab in the primary color", () => {
      const { container } = render(<Diagram def={PILL} state={ZERO_CALC_STATE} />);
      const tabs = [...container.querySelectorAll('rect[height="12"]')];
      expect(tabs).toHaveLength(4);
      for (const tab of tabs) expect(tab.getAttribute("stroke")).toBe(PRIMARY);
    });
  });
});
