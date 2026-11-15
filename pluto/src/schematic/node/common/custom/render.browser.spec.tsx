// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { computed, resolve } from "@synnaxlabs/lyra/styletest";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, location, type xy } from "@synnaxlabs/x";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
} from "react";
import { describe, expect, it } from "vitest";
import { render, renderHook } from "vitest-browser-react";

import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";

const STROKE = color.construct("#cfd1d4");
const FILL = color.construct("#ffffff");

const SPEC: schematic.symbol.Spec = {
  svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect class="main" width="10" height="10" stroke="red" fill="blue"/></svg>',
  states: [
    {
      key: "base",
      name: "Base",
      regions: [
        {
          key: "main",
          name: "Main",
          selectors: [".main"],
          strokeColor: STROKE,
          fillColor: FILL,
        },
      ],
    },
  ],
  variant: "static",
  handles: [],
  scale: 1,
  strokeScaled: false,
  previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
};

interface Paint {
  stroke: string;
  fill: string;
}

const renderCustom = async (theme: Theming.ThemeSpec): Promise<Paint> => {
  const Themed = ({ children }: PropsWithChildren): ReactElement => (
    <Theming.Provider theme={theme}>{children}</Theming.Provider>
  );
  const { result } = await renderHook(
    () =>
      Custom.useRender({
        orientation: "left",
        activeState: "base",
        externalScale: 1,
        spec: SPEC,
      }),
    { wrapper: Themed },
  );
  const container = document.createElement("div");
  document.body.appendChild(container);
  result.current(container);
  const rect = container.querySelector("rect");
  if (rect == null) throw new Error("custom symbol not mounted");
  const paint = { stroke: computed(rect, "stroke"), fill: computed(rect, "fill") };
  container.remove();
  return paint;
};

const renderBuiltIn = async (theme: Theming.ThemeSpec): Promise<Element> => {
  const { container } = await render(
    <Theming.Provider theme={theme}>
      <Primitive.SVG dimensions={{ width: 10, height: 10 }} strokeColor={STROKE}>
        <rect width="10" height="10" />
      </Primitive.SVG>
    </Theming.Provider>,
  );
  const rect = container.querySelector("rect");
  if (rect == null) throw new Error("built-in symbol not mounted");
  return rect;
};

describe("Custom.useRender theme band", () => {
  describe.each([
    ["light", Theming.SYNNAX_LIGHT],
    ["dark", Theming.SYNNAX_DARK],
  ])("%s theme", (_, theme) => {
    it("should paint a region stroke the same as a built-in symbol stroke", async () => {
      const builtIn = computed(await renderBuiltIn(theme), "stroke");
      const { stroke: custom } = await renderCustom(theme);
      expect(custom).not.toBe(resolve(color.hex(STROKE), "stroke"));
      expect(custom).toBe(builtIn);
    });

    it("should paint a region fill as stored", async () => {
      const { fill } = await renderCustom(theme);
      expect(fill).toBe(resolve(color.hex(FILL), "fill"));
    });
  });
});

const STALE = color.construct("#e5a50a");

const COLORED_SPEC: schematic.symbol.Spec = {
  ...SPEC,
  svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect class="stroked" width="10" height="10" stroke="red"/><rect class="unstroked" width="5" height="5" stroke="none" fill="blue"/></svg>',
  states: [
    {
      key: "base",
      name: "Base",
      regions: [
        {
          key: "stroked",
          name: "Stroked",
          selectors: [".stroked"],
          strokeColor: STROKE,
        },
        {
          key: "unstroked",
          name: "Unstroked",
          selectors: [".unstroked"],
          strokeColor: color.TRANSPARENT,
          fillColor: FILL,
        },
      ],
    },
  ],
};

interface ColoredProps {
  strokeColor?: color.Crude;
}

const renderColored = async (strokeColor?: color.Crude) => {
  const { result, rerender } = await renderHook(
    (props?: ColoredProps) =>
      Custom.useRender({
        orientation: "left",
        activeState: "base",
        externalScale: 1,
        spec: COLORED_SPEC,
        strokeColor: props?.strokeColor,
      }),
    { initialProps: { strokeColor } },
  );
  const { container } = await render(
    <Theming.Provider theme={Theming.SYNNAX_DARK}>
      <Primitive.Div ref={result.current} />
    </Theming.Provider>,
  );
  const query = (selector: string): Element => {
    const el = container.querySelector(selector);
    if (el == null) throw new Error(`${selector} not mounted`);
    return el;
  };
  return {
    svg: query("svg"),
    stroked: query(".stroked"),
    unstroked: query(".unstroked"),
    rerender,
  };
};

const renderColoredBuiltIn = async (strokeColor: color.Crude) => {
  const { container } = await render(
    <Theming.Provider theme={Theming.SYNNAX_DARK}>
      <Primitive.SVG dimensions={{ width: 10, height: 10 }} strokeColor={strokeColor}>
        <rect width="10" height="10" />
      </Primitive.SVG>
    </Theming.Provider>,
  );
  const svg = container.querySelector("svg");
  const rect = container.querySelector("rect");
  if (svg == null || rect == null) throw new Error("built-in symbol not mounted");
  return { svg, rect };
};

describe("Custom.useRender symbol color", () => {
  it("should paint the root svg the same as a built-in symbol", async () => {
    const builtIn = await renderColoredBuiltIn(STALE);
    const { svg } = await renderColored(STALE);
    expect(computed(svg, "stroke")).toBe(computed(builtIn.svg, "stroke"));
  });

  it("should paint a stroked region the same as a built-in symbol", async () => {
    const builtIn = await renderColoredBuiltIn(STALE);
    const { stroked } = await renderColored(STALE);
    expect(computed(stroked, "stroke")).toBe(computed(builtIn.rect, "stroke"));
  });

  it("should keep an unstroked region unstroked", async () => {
    const plain = await renderColored();
    const { unstroked } = await renderColored(STALE);
    expect(computed(unstroked, "stroke")).toBe(computed(plain.unstroked, "stroke"));
  });

  it("should restore the region colors when the symbol color clears", async () => {
    const builtIn = await renderColoredBuiltIn(STROKE);
    const { stroked, rerender } = await renderColored(STALE);
    await rerender({ strokeColor: undefined });
    expect(computed(stroked, "stroke")).toBe(computed(builtIn.rect, "stroke"));
  });
});

const BODY = '<rect width="40" height="20"/>';
const MARKER = '<rect class="marker" width="10" height="10"/>';

// Markup of a 40 by 20 drawing with a 10 by 10 marker in its top left corner.
const DRAWINGS: [string, string][] = [
  ["shapes", `${BODY}${MARKER}`],
  ["shapes beside a group", `<g>${BODY}</g>${MARKER}`],
  ["shapes with a group only in defs", `<defs><g>${BODY}</g></defs>${MARKER}`],
  [
    "a moved group",
    `${BODY}<g transform="translate(10 10)"><rect class="marker" x="-10" y="-10" width="10" height="10"/></g>`,
  ],
];

const createOrientedSpec = (drawing: string): schematic.symbol.Spec => ({
  ...SPEC,
  svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20">${drawing}</svg>`,
  states: [],
});

// Where the marker's center lands as a fraction of the symbol's box, turned as a
// built-in symbol turns.
const MARKER_CENTERS: Record<location.Outer, xy.XY> = {
  left: { x: 0.125, y: 0.25 },
  right: { x: 0.875, y: 0.75 },
  top: { x: 0.25, y: 0.875 },
  bottom: { x: 0.75, y: 0.125 },
};

interface ContainerProps {
  ref: RefCallback<HTMLElement>;
  orientation: location.Outer;
}

const CONTAINERS: [string, FC<ContainerProps>][] = [
  ["static", Primitive.Div],
  ["actuator", Toggle.Button],
];

const renderMarkerCenter = async (
  spec: schematic.symbol.Spec,
  Container: FC<ContainerProps>,
  orientation: location.Outer,
): Promise<xy.XY> => {
  const { result } = await renderHook(() =>
    Custom.useRender({
      orientation,
      activeState: "base",
      externalScale: 1,
      spec,
    }),
  );
  const { container } = await render(
    <Container ref={result.current} orientation={orientation} />,
  );
  const svg = container.querySelector("svg");
  const marker = container.querySelector(".marker");
  if (svg == null || !(marker instanceof SVGGraphicsElement))
    throw new Error("custom symbol not mounted");
  const matrix = marker.getScreenCTM();
  if (matrix == null) throw new Error("marker has no screen matrix");
  const { x, y, width, height } = marker.getBBox();
  const center = new DOMPoint(x + width / 2, y + height / 2).matrixTransform(matrix);
  const box = svg.getBoundingClientRect();
  return {
    x: (center.x - box.left) / box.width,
    y: (center.y - box.top) / box.height,
  };
};

describe("Custom.useRender orientation", () => {
  describe.each(DRAWINGS)("%s", (_, drawing) => {
    const spec = createOrientedSpec(drawing);
    describe.each(CONTAINERS)("%s symbol", (_, Container) => {
      it.each(location.OUTER_LOCATIONS)(
        "should turn the drawing to %s",
        async (orientation) => {
          const center = await renderMarkerCenter(spec, Container, orientation);
          expect(center.x).toBeCloseTo(MARKER_CENTERS[orientation].x, 3);
          expect(center.y).toBeCloseTo(MARKER_CENTERS[orientation].y, 3);
        },
      );
    });
  });
});
