// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";
import "@xyflow/react/dist/base.css";

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { type color, type location, narrow } from "@synnaxlabs/x";
import { type Node as RFNode, type NodeProps, ReactFlow } from "@xyflow/react";
import type { FC, ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { server } from "vitest/browser";

import { CSS } from "@/css";
import { Node } from "@/schematic/node";

const ORIENTATIONS: location.Outer[] = ["left", "right", "top", "bottom"];
const SCALES = [1, 2];
const COLOR = "#3e8ef7";
// General widgets, tanks, and cylinders draw from their config, not a fixed drawing.
const EXCLUDED_GROUP = "general";
const EXCLUDED_VARIANTS = new Set(["tank", "cylinder"]);
const SHAPES = "path, rect, circle, ellipse, line, polyline, polygon, text";
const GEOMETRY = [
  "d",
  "x",
  "y",
  "width",
  "height",
  "rx",
  "ry",
  "cx",
  "cy",
  "r",
  "x1",
  "y1",
  "x2",
  "y2",
  "points",
  "transform",
];
const PAINT = [
  "stroke",
  "fill",
  "stroke-width",
  "stroke-linecap",
  "stroke-dasharray",
  "vector-effect",
];
const TEXT = ["font", "text-anchor", "dominant-baseline"];

interface DrawingProps {
  orientation: location.Outer;
  scale: number;
  strokeColor?: color.Crude;
  enabled?: boolean;
}

interface Case extends Record<string, unknown> {
  variant: string;
  name: string;
  props: DrawingProps;
}

interface Drawn {
  box: string;
  handles: Record<string, string>;
  shapes: string[];
}

interface Fingerprint {
  root: string;
  geometry: string[];
  cases: Record<string, Drawn>;
}

const FAMILIES = Node.GROUPS.filter(({ key }) => key !== EXCLUDED_GROUP).map(
  ({ key, symbols }) => ({
    key,
    variants: symbols.filter((v) => !EXCLUDED_VARIANTS.has(v)),
  }),
);

const createCases = (variant: string, enabled?: true): Case[] =>
  ORIENTATIONS.flatMap((orientation) =>
    SCALES.flatMap((scale) =>
      [undefined, COLOR].map((strokeColor) => ({
        variant,
        name: [orientation, `x${scale}`, enabled && "on", strokeColor && "colored"]
          .filter(Boolean)
          .join(" "),
        props: { orientation, scale, strokeColor, enabled },
      })),
    ),
  );

const Drawing = ({
  data: { variant, props },
}: NodeProps<RFNode<Case>>): ReactElement => {
  const Preview = Node.resolveSpec(variant).Preview as FC<DrawingProps>;
  return <Preview {...props} />;
};

const NODE_TYPES = { drawing: Drawing };

const round = (value: number): number => Math.round(value * 100) / 100;

// A computed color has more digits than its math keeps the same on each platform.
const roundAll = (text: string): string =>
  text.replace(/\d+\.\d+/g, (n) => round(Number(n)).toString());

/** @returns the left, top, width, and height of the element from the origin. */
const box = (el: Element, origin: DOMRect): string => {
  const { left, top, width, height } = el.getBoundingClientRect();
  return [left - origin.left, top - origin.top, width, height].map(round).join(" ");
};

const center = (el: Element, origin: DOMRect): string => {
  const { left, top, width, height } = el.getBoundingClientRect();
  const x = left + width / 2 - origin.left;
  return [x, top + height / 2 - origin.top].map(round).join(" ");
};

const geometry = (el: Element): string =>
  [
    el.tagName,
    ...GEOMETRY.filter((a) => el.hasAttribute(a)).map(
      (a) => `${a}=${el.getAttribute(a)}`,
    ),
    el.textContent,
  ]
    .filter(Boolean)
    .join(" ");

// Font hinting changes glyph widths on each platform, so text records its anchor and
// the properties that place and size its glyphs, not its box.
const place = (el: Element, origin: DOMRect, style: CSSStyleDeclaration): string => {
  if (!(el instanceof SVGTextElement)) return box(el, origin);
  const matrix = el.getScreenCTM();
  if (matrix == null) throw new Error("text has no screen matrix");
  const { x, y } = new DOMPoint(
    el.x.baseVal.getItem(0).value,
    el.y.baseVal.getItem(0).value,
  ).matrixTransform(matrix);
  const anchor = [x - origin.left, y - origin.top].map(round).join(" ");
  return [anchor, ...TEXT.map((p) => style.getPropertyValue(p))].join(" ");
};

const paint = (el: Element, origin: DOMRect): string => {
  const style = getComputedStyle(el);
  const values = PAINT.map((p) => roundAll(style.getPropertyValue(p)));
  return [place(el, origin, style), ...values].join(" | ");
};

const measure = (node: Element): Omit<Fingerprint, "cases"> & { drawn: Drawn } => {
  const origin = node.getBoundingClientRect();
  const svg = node.querySelector("svg");
  if (svg == null || node.firstElementChild == null)
    throw new Error(`node ${node.getAttribute("data-id")} drew nothing`);
  const shapes = Array.from(svg.querySelectorAll(SHAPES));
  const handles = Array.from(node.querySelectorAll(".react-flow__handle"), (h) => [
    h.getAttribute("data-handleid"),
    `${h.getAttribute("data-handlepos")} ${center(h, origin)}`,
  ]);
  return {
    root: node.firstElementChild.tagName.toLowerCase(),
    geometry: shapes.map(geometry),
    drawn: {
      box: box(svg, origin),
      handles: Object.fromEntries(handles),
      shapes: shapes.map((s) => paint(s, origin)),
    },
  };
};

/** Draws each case as a node of one flow and adds its fingerprint to the record. */
const draw = async (cases: Case[], record: Record<string, Fingerprint>) => {
  const nodes = cases.map((data, i) => ({
    id: i.toString(),
    type: "drawing",
    position: { x: 0, y: 0 },
    data,
  }));
  const { container, unmount } = await renderThemed(
    <div style={{ width: 800, height: 800 }}>
      <ReactFlow
        className={CSS.B("schematic")}
        nodes={nodes}
        nodeTypes={NODE_TYPES}
        defaultViewport={{ x: 0, y: 0, zoom: 1 }}
      />
    </div>,
  );
  const rendered = await vi.waitFor(() => {
    const els = container.querySelectorAll(".react-flow__node");
    expect(els).toHaveLength(cases.length);
    return els;
  });
  rendered.forEach((node) => {
    const { variant, name } = cases[Number(node.getAttribute("data-id"))];
    const { drawn, ...rest } = measure(node);
    record[variant] ??= { ...rest, cases: {} };
    record[variant].cases[name] = drawn;
  });
  await unmount();
};

const sortKeys = (_: string, value: unknown): unknown =>
  narrow.isObject(value)
    ? Object.fromEntries(Object.entries(value).sort(([a], [b]) => (a < b ? -1 : 1)))
    : value;

describe("Drawings", () => {
  it.each(FAMILIES)(
    "should draw each $key variant as recorded",
    async ({ key, variants }) => {
      const record: Record<string, Fingerprint> = {};
      await draw(
        variants.flatMap((v) => createCases(v)),
        record,
      );
      const toggles = variants.filter((v) => record[v].root === "button");
      await draw(
        toggles.flatMap((v) => createCases(v, true)),
        record,
      );
      await expect(`${JSON.stringify(record, sortKeys, 2)}\n`).toMatchFileSnapshot(
        `__snapshots__/drawings/${key}.${server.browser}.json`,
      );
    },
  );
});
