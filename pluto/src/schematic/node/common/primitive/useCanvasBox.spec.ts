// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";

interface Layout {
  zoom: number;
  // offset is where the element's border box starts in the root, in layout units.
  offset: xy.XY;
  border: number;
  padding: { x: number; y: number };
}

const ROOT_WIDTH = 200;
const ROOT_SCREEN = xy.construct(300, 100);
const WIDTH = 80;
const HEIGHT = 30;

const rect = (origin: xy.XY, width: number, height: number): DOMRect =>
  new DOMRect(origin.x, origin.y, width, height);

const define = (el: HTMLElement, props: Record<string, number>): void => {
  for (const [key, value] of Object.entries(props))
    Object.defineProperty(el, key, { configurable: true, value });
};

// jsdom does no layout, so each element reports the sizes a browser would give it under
// a diagram zoomed by a CSS transform: client rects scale and layout sizes do not.
const createSymbol = ({ zoom, offset, border, padding }: Layout): HTMLElement => {
  const root = document.createElement("div");
  root.className = CSS.B("symbol-primitive");
  root.getBoundingClientRect = () => rect(ROOT_SCREEN, ROOT_WIDTH * zoom, 50 * zoom);
  define(root, { offsetWidth: ROOT_WIDTH });
  const el = document.createElement("div");
  el.style.width = `${WIDTH}px`;
  el.style.height = `${HEIGHT}px`;
  el.style.padding = `${padding.y}px ${padding.x}px`;
  el.getBoundingClientRect = () =>
    rect(
      xy.translate(ROOT_SCREEN, xy.scale(offset, zoom)),
      WIDTH * zoom,
      HEIGHT * zoom,
    );
  define(el, {
    offsetWidth: WIDTH,
    offsetHeight: HEIGHT,
    clientLeft: border,
    clientTop: border,
    clientWidth: WIDTH - 2 * border,
    clientHeight: HEIGHT - 2 * border,
  });
  root.appendChild(el);
  document.body.appendChild(root);
  return el;
};

const measure = async (el: HTMLElement, position?: xy.XY): Promise<box.Box> => {
  const { result } = renderHook(() => Primitive.useCanvasBox(position));
  act(() => {
    result.current.ref(el);
  });
  await waitFor(() => expect(box.areaIsZero(result.current.box)).toBe(false));
  return result.current.box;
};

const LAYOUT: Layout = {
  zoom: 1,
  offset: xy.construct(20, 10),
  border: 1,
  padding: { x: 12, y: 2 },
};

// The content box sits inside the border and padding.
const CONTENT = box.construct(
  { x: 20 + 1 + 12, y: 10 + 1 + 2 },
  { width: WIDTH - 2 - 24, height: HEIGHT - 2 - 4 },
);

describe("Primitive.useCanvasBox", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("should be empty before the element mounts", () => {
    const { result } = renderHook(() => Primitive.useCanvasBox());
    expect(result.current.box).toEqual(box.ZERO);
  });

  it("should measure the content box inside the border and padding", async () => {
    expect(await measure(createSymbol(LAYOUT))).toEqual(CONTENT);
  });

  it("should measure in diagram units at any zoom", async () => {
    expect(await measure(createSymbol({ ...LAYOUT, zoom: 2.5 }))).toEqual(CONTENT);
  });

  it("should be empty while the symbol is hidden", async () => {
    const el = createSymbol(LAYOUT);
    const { result } = renderHook(() => Primitive.useCanvasBox());
    act(() => {
      result.current.ref(el);
    });
    await waitFor(() => expect(result.current.box).toEqual(CONTENT));
    const root = el.parentElement as HTMLElement;
    root.getBoundingClientRect = () => rect(xy.ZERO, 0, 0);
    define(root, { offsetWidth: 0 });
    el.getBoundingClientRect = () => rect(xy.ZERO, 0, 0);
    el.style.width = "0px";
    el.style.height = "0px";
    define(el, { offsetWidth: 0, offsetHeight: 0, clientWidth: 0, clientHeight: 0 });
    await waitFor(() => expect(result.current.box).toEqual(box.ZERO));
  });

  // A field above a readout can grow without resizing the readout's cell.
  it("should measure again when the element moves without resizing", async () => {
    const el = createSymbol(LAYOUT);
    const { result } = renderHook(() => Primitive.useCanvasBox());
    act(() => {
      result.current.ref(el);
    });
    await waitFor(() => expect(result.current.box).toEqual(CONTENT));
    const shift = xy.construct(0, 4);
    const moved = xy.translate(xy.translate(ROOT_SCREEN, LAYOUT.offset), shift);
    el.getBoundingClientRect = () => rect(moved, WIDTH, HEIGHT);
    (el.parentElement as HTMLElement).style.height = "54px";
    await waitFor(() =>
      expect(result.current.box).toEqual(box.translate(CONTENT, shift)),
    );
  });

  it("should throw for an element outside a symbol", () => {
    const { result } = renderHook(() => Primitive.useCanvasBox());
    expect(() => result.current.ref(document.createElement("div"))).toThrow(
      "useCanvasBox needs an element inside a symbol primitive",
    );
  });

  it("should place the box at the node's position", async () => {
    const position = xy.construct(400, -50);
    expect(await measure(createSymbol(LAYOUT), position)).toEqual(
      box.translate(CONTENT, position),
    );
  });
});
