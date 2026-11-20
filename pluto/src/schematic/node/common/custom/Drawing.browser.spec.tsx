// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { computed, resolve } from "@synnaxlabs/lyra/styletest";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type dimensions, location, type xy } from "@synnaxlabs/x";
import type { FC, ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { CSS } from "@/css";
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
  const { container } = await render(
    <Theming.Provider theme={theme}>
      <Primitive.Div>
        <Custom.Drawing spec={SPEC} />
      </Primitive.Div>
    </Theming.Provider>,
  );
  const rect = container.querySelector("rect");
  if (rect == null) throw new Error("custom symbol not mounted");
  return { stroke: computed(rect, "stroke"), fill: computed(rect, "fill") };
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

describe("Custom.Drawing theme band", () => {
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

// Shapes whose strokes show: by a region, by their own paint in a region with no
// stroke, and by their own paint outside every region.
const SHOWN = [".stroked", ".unset", ".loose"];
// Shapes whose strokes do not show: by a region, with no stroke of their own, and
// outside the drawn tree.
const HIDDEN = [".unstroked", ".bare", ".template"];

const COLORED_SPEC: schematic.symbol.Spec = {
  ...SPEC,
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
    '<rect class="stroked" width="10" height="10" stroke="red"/>' +
    '<rect class="unstroked" width="5" height="5" stroke="none" fill="blue"/>' +
    '<rect class="unset" width="4" height="4" stroke="red"/>' +
    '<rect class="loose" width="3" height="3" stroke="red"/>' +
    '<rect class="bare" width="2" height="2" fill="blue"/>' +
    '<defs><rect class="template" width="1" height="1" stroke="red"/></defs>' +
    "</svg>",
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
        { key: "unset", name: "Unset", selectors: [".unset"] },
      ],
    },
  ],
};

const DISPLAY = "--pluto-symbol-display";

const ColoredDrawing = ({
  strokeColor,
}: {
  strokeColor?: color.Crude;
}): ReactElement => (
  <Theming.Provider theme={Theming.SYNNAX_DARK}>
    <Primitive.Div>
      <Custom.Drawing spec={COLORED_SPEC} strokeColor={strokeColor} />
    </Primitive.Div>
  </Theming.Provider>
);

const renderColored = async (strokeColor?: color.Crude) => {
  const { container, rerender } = await render(
    <ColoredDrawing strokeColor={strokeColor} />,
  );
  const query = (selector: string): Element => {
    const el = container.querySelector(selector);
    if (el == null) throw new Error(`${selector} not mounted`);
    return el;
  };
  return {
    stroke: (selector: string): string => computed(query(selector), "stroke"),
    display: (): string => computed(query("svg"), DISPLAY),
    rerender: async (next?: color.Crude) => {
      await rerender(<ColoredDrawing strokeColor={next} />);
    },
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
  return { display: computed(svg, DISPLAY), stroke: computed(rect, "stroke") };
};

describe("Custom.Drawing symbol color", () => {
  it("should give the root svg the display color of a built-in symbol", async () => {
    const builtIn = await renderColoredBuiltIn(STALE);
    const { display } = await renderColored(STALE);
    expect(display()).toBe(builtIn.display);
  });

  it.each(SHOWN)("should paint the %s stroke as a built-in symbol", async (shape) => {
    const builtIn = await renderColoredBuiltIn(STALE);
    const { stroke } = await renderColored(STALE);
    expect(stroke(shape)).toBe(builtIn.stroke);
  });

  it.each(HIDDEN)("should keep the %s stroke hidden", async (shape) => {
    const plain = await renderColored();
    const { stroke } = await renderColored(STALE);
    expect(stroke(shape)).toBe(plain.stroke(shape));
  });

  it.each([...SHOWN, ...HIDDEN, "svg"])(
    "should restore the %s stroke when the symbol color clears",
    async (shape) => {
      const plain = await renderColored();
      const { stroke, rerender } = await renderColored(STALE);
      await rerender(undefined);
      expect(stroke(shape)).toBe(plain.stroke(shape));
    },
  );

  it("should not stroke a shape with no stroke of its own", async () => {
    const { stroke } = await renderColored();
    expect(stroke(".bare")).toBe("none");
  });
});

const ROOT_PAINT_SPEC: schematic.symbol.Spec = {
  ...SPEC,
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" fill="red" stroke="blue">' +
    '<rect class="bare" width="10" height="10"/></svg>',
  states: [],
};

const ROOT_PAINT_SYMBOLS: [string, ReactElement][] = [
  [
    "static symbol",
    <Primitive.Div key="static">
      <Custom.Drawing spec={ROOT_PAINT_SPEC} />
    </Primitive.Div>,
  ],
  [
    "enabled actuator",
    <Toggle.Button key="actuator" enabled>
      <Custom.Drawing spec={ROOT_PAINT_SPEC} enabled />
    </Toggle.Button>,
  ],
];

describe("Custom.Drawing root paint", () => {
  it.each(ROOT_PAINT_SYMBOLS)(
    "should paint the shapes of a %s as the symbol paints its root",
    async (_, symbol) => {
      const { container } = await render(
        <Theming.Provider theme={Theming.SYNNAX_DARK}>{symbol}</Theming.Provider>,
      );
      const svg = container.querySelector("svg");
      const shape = container.querySelector(".bare");
      if (svg == null || shape == null) throw new Error("custom symbol not mounted");
      const paint = (el: Element) => {
        const { fill, stroke } = getComputedStyle(el);
        return { fill, stroke };
      };
      expect(paint(shape)).toEqual(paint(svg));
    },
  );
});

const HOLD_DELAY = 10_000;
const HOLD_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
  '<rect width="10" height="10" stroke="red" fill="none"/></svg>';

const BUILT_IN_HOLD = (
  <Toggle.Button onClickDelay={HOLD_DELAY}>
    <Primitive.SVG dimensions={{ width: 10, height: 10 }} strokeColor={STALE}>
      <rect width="10" height="10" />
    </Primitive.SVG>
  </Toggle.Button>
);

const renderHeld = async (symbol: ReactElement): Promise<Element> => {
  const { container } = await render(
    <Theming.Provider theme={Theming.SYNNAX_DARK}>{symbol}</Theming.Provider>,
  );
  const button = container.querySelector("button");
  if (button == null) throw new Error("toggle not rendered");
  button.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
  await expect.poll(() => button.classList.contains("pluto--pressed")).toBe(true);
  return button;
};

// The selector matches any first rect, so it would reach a fill beside the shapes.
const HOLD_SPEC: schematic.symbol.Spec = {
  ...SPEC,
  svg: HOLD_SVG,
  states: [
    {
      key: "base",
      name: "Base",
      regions: [
        {
          key: "main",
          name: "Main",
          selectors: ["rect:nth-of-type(1)"],
          strokeColor: STROKE,
        },
      ],
    },
  ],
};

const renderCustomHeld = async (): Promise<Element> =>
  await renderHeld(
    <Toggle.Button className={CSS.BM("symbol", "custom")} onClickDelay={HOLD_DELAY}>
      <Custom.Drawing spec={HOLD_SPEC} strokeColor={STALE} />
    </Toggle.Button>,
  );

const findHoldFill = (button: Element): SVGGraphicsElement => {
  const fill = button.querySelector(".pluto-symbol-hold__fill");
  if (!(fill instanceof SVGGraphicsElement)) throw new Error("hold fill not drawn");
  return fill;
};

interface HoldFillPaint {
  display: string;
  animation: string;
  fill: string;
  box: xy.XY & dimensions.Dimensions;
}

const readHoldFill = (button: Element): HoldFillPaint => {
  const fill = findHoldFill(button);
  const { x, y, width, height } = fill.getBBox();
  return {
    display: computed(fill, "display"),
    animation: computed(fill, "animation-name"),
    fill: computed(fill, "fill"),
    box: { x, y, width, height },
  };
};

// The element the fill's mask draws, so the fill paints only inside its shapes.
const findMaskSource = (fill: Element): Element | null => {
  const doc = fill.ownerDocument;
  const maskID = /url\(#(.+)\)/.exec(fill.getAttribute("mask") ?? "")?.[1] ?? "";
  const href = doc.getElementById(maskID)?.querySelector("use")?.getAttribute("href");
  return doc.getElementById(href?.slice(1) ?? "");
};

describe("Custom.Drawing hold fill", () => {
  it("should fill the drawing while held, as a built-in symbol does", async () => {
    const builtIn = readHoldFill(await renderHeld(BUILT_IN_HOLD));
    const button = await renderCustomHeld();
    expect(readHoldFill(button)).toEqual(builtIn);
    const source = findMaskSource(findHoldFill(button));
    expect(source?.querySelector("rect[stroke=red]")).not.toBeNull();
  });

  it("should not draw a box over a held custom actuator", async () => {
    const builtIn = await renderHeld(BUILT_IN_HOLD);
    const custom = await renderCustomHeld();
    expect(getComputedStyle(custom, "::after").content).toBe(
      getComputedStyle(builtIn, "::after").content,
    );
  });
});

const BUILT_IN_TOGGLE_SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
    '<rect class="shape" width="10" height="10"/></svg>',
  handles: [],
});

const JSX_TOGGLE_DRAWING = (
  <Primitive.SVG dimensions={{ width: 10, height: 10 }}>
    <rect className="shape" width="10" height="10" />
  </Primitive.SVG>
);

const press = async (button: Element): Promise<void> => {
  button.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
  await expect.poll(() => button.classList.contains("pluto--pressed")).toBe(true);
};

const TOGGLE_STATES: [string, boolean, (button: Element) => Promise<void>][] = [
  ["hovered", false, userEvent.hover],
  ["hovered enabled", true, userEvent.hover],
  ["pressed", false, press],
  ["pressed enabled", true, press],
];

const renderToggleShape = async (
  drawing: ReactElement,
  enabled: boolean,
  interact: (button: Element) => Promise<void>,
): Promise<{ rest: Paint; changed: Paint }> => {
  const { container } = await render(
    <Theming.Provider theme={Theming.SYNNAX_DARK}>
      <div className={CSS.BE("symbol", "container")}>
        <Toggle.Button enabled={enabled} onClickDelay={HOLD_DELAY}>
          {drawing}
        </Toggle.Button>
      </div>
    </Theming.Provider>,
  );
  const button = container.querySelector("button");
  const shape = container.querySelector(".shape");
  if (button == null || shape == null) throw new Error("toggle not rendered");
  const paint = (): Paint => ({
    stroke: computed(shape, "stroke"),
    fill: computed(shape, "fill"),
  });
  const rest = paint();
  await interact(button);
  const changed = paint();
  await userEvent.unhover(button);
  return { rest, changed };
};

describe("Custom.Drawing built-in toggle", () => {
  it.each(TOGGLE_STATES)(
    "should paint the shapes of a %s toggle as a JSX toggle",
    async (_, enabled, interact) => {
      const jsx = await renderToggleShape(JSX_TOGGLE_DRAWING, enabled, interact);
      const builtIn = await renderToggleShape(
        <Custom.Drawing spec={BUILT_IN_TOGGLE_SPEC} builtIn enabled={enabled} />,
        enabled,
        interact,
      );
      expect(jsx.changed.stroke).not.toBe(jsx.rest.stroke);
      expect(builtIn).toEqual(jsx);
    },
  );
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
  orientation: location.Outer;
  children: ReactElement;
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
  const { container } = await render(
    <Container orientation={orientation}>
      <Custom.Drawing spec={spec} orientation={orientation} />
    </Container>,
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

describe("Custom.Drawing orientation", () => {
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

describe("Custom.Drawing size", () => {
  const spec = { ...createOrientedSpec(BODY), scale: 2 };

  it.each([
    ["left", { width: 120, height: 60 }],
    ["top", { width: 60, height: 120 }],
  ] as const)(
    "should size the drawing by both scales when it faces %s",
    async (orientation, size) => {
      const { container } = await render(
        <Primitive.Div orientation={orientation}>
          <Custom.Drawing spec={spec} orientation={orientation} scale={1.5} />
        </Primitive.Div>,
      );
      const svg = container.querySelector("svg");
      if (svg == null) throw new Error("custom symbol not mounted");
      const { width, height } = svg.getBoundingClientRect();
      expect({ width, height }).toEqual(size);
    },
  );
});
