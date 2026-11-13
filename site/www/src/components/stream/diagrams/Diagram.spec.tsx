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
import { Diagram, type DiagramDef } from "@/components/stream/diagrams";

const PRIMARY = "var(--pluto-primary-p1)";
const ERROR = "var(--pluto-error-z)";
const DIM = "var(--pluto-gray-l5)";

type NodeDef = DiagramDef["nodes"][number];

const icon = (id: string): NodeDef["icon"] => {
  const Icon = ({ style }: { style?: CSSProperties }): ReactElement => (
    <svg data-testid={id} style={style} />
  );
  return Icon;
};

const node = (id: string, x: number, y = 100): NodeDef => ({
  id,
  x,
  y,
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

// The trace from "a" starts at x = 180. It ends at x = endX, at the left tab of "b".
const edge = (fromY: number, toY: number, endX = 280): DiagramDef => ({
  viewBox: "0 0 900 200",
  nodes: [node("a", 100, fromY), node("b", endX + 80, toY)],
  edges: [{ from: "a", to: "b" }],
});

const path = (def: DiagramDef): string | null =>
  traces(
    render(<Diagram def={def} state={ZERO_CALC_STATE} />).container,
  )[0].getAttribute("d");

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

    it("should treat a drop under one unit as the same height", () => {
      expect(path(edge(50, 50.5))).toBe("M180,50 L280,50.5");
    });

    it("should turn down at the horizontal midpoint with rounded corners", () => {
      expect(path(edge(0, 100))).toBe(
        "M180,0 L222,0 Q230,0 230,8 L230,92 Q230,100 238,100 L280,100",
      );
    });

    it("should turn up when the end is above the start", () => {
      expect(path(edge(100, 0))).toBe(
        "M180,100 L222,100 Q230,100 230,92 L230,8 Q230,0 238,0 L280,0",
      );
    });

    it("should shrink the corners to half of a short drop", () => {
      expect(path(edge(0, 6))).toBe(
        "M180,0 L227,0 Q230,0 230,3 L230,3 Q230,6 233,6 L280,6",
      );
    });

    it("should shrink the corners to half of the run to the midpoint", () => {
      expect(path(edge(0, 100, 192))).toBe(
        "M180,0 L183,0 Q186,0 186,3 L186,97 Q186,100 189,100 L192,100",
      );
    });

    it("should cross out an excluded trace at its midpoint", () => {
      const { container } = render(
        <Diagram
          def={edge(0, 100)}
          state={{ ...ZERO_CALC_STATE, excludedNodes: ["a"] }}
        />,
      );
      expect(container.querySelector("g[transform]")!.getAttribute("transform")).toBe(
        "translate(230,50)",
      );
    });

    it.each([
      ["dim an idle trace", [], [], "var(--pluto-gray-l2)"],
      [
        "color a flowing trace in the primary color",
        ["src"],
        [],
        "var(--pluto-primary-z-15)",
      ],
      ["rank an exclusion above flow", ["src"], ["src"], "var(--pluto-error-z-15)"],
    ])("should %s", (_, activeNodes, excludedNodes, color) => {
      const { container } = render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, activeNodes, excludedNodes }}
        />,
      );
      expect(traces(container)[0].getAttribute("stroke")).toBe(color);
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

    it("should color an alarm node in the error color even when it is active", () => {
      render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, activeNodes: ["src"], alarmNodes: ["src"] }}
        />,
      );
      expect(screen.getByTestId("src").style.color).toBe(ERROR);
    });

    it("should color an excluded node in the error color even when it is active", () => {
      render(
        <Diagram
          def={DEF}
          state={{ ...ZERO_CALC_STATE, activeNodes: ["mid"], excludedNodes: ["mid"] }}
        />,
      );
      expect(screen.getByTestId("mid").style.color).toBe(ERROR);
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
