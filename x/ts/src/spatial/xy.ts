// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import {
  clientXY,
  dimensions,
  numberCouple,
  signedDimension,
  xy,
  type NumberCouple,
  type ClientXY,
} from "@/spatial/base";
import type * as direction from "@/spatial/direction";

export { clientXY, xy, type ClientXY as Client };

export type XY = z.infer<typeof xy>;
export const crude = z.union([
  xy,
  numberCouple,
  dimensions,
  signedDimension,
  clientXY,
]);
export type Crude = z.infer<typeof crude>;

export const construct = (x: Crude | number, y?: number): XY => {
  if (typeof x === "number") return { x, y: y ?? x };
  if (Array.isArray(x)) return { x: x[0], y: x[1] };
  if ("signedWidth" in x) return { x: x.signedWidth, y: x.signedHeight };
  if ("clientX" in x) return { x: x.clientX, y: x.clientY };
  if ("width" in x) return { x: x.width, y: x.height };
  return { ...x };
};

/** An x and y coordinate of zero */
export const ZERO = { x: 0, y: 0 };

/** An x and y coordinate of one */
export const ONE = { x: 1, y: 1 };

/** An x and y coordinate of infinity */
export const INFINITY = { x: Infinity, y: Infinity };

export const NAN = { x: NaN, y: NaN };

export const equals = (a: Crude, b: Crude): boolean => {
  const a_ = construct(a);
  const b_ = construct(b);
  return a_.x === b_.x && a_.y === b_.y;
};

export const isZero = (c: Crude): boolean => equals(c, ZERO);

export const scale = (c: Crude, s: number): XY => {
  const p = construct(c);
  return { x: p.x * s, y: p.y * s };
};

export const translateX = (c: Crude, x: number): XY => {
  const p = construct(c);
  return { x: p.x + x, y: p.y };
};

export const translateY = (c: Crude, y: number): XY => {
  const p = construct(c);
  return { x: p.x, y: p.y + y };
};

export const translate = (...cb: Crude[]): XY =>
  cb.reduce((p: XY, c) => {
    const xy = construct(c);
    return { x: p.x + xy.x, y: p.y + xy.y };
  }, ZERO);

export const set = (c: Crude, direction: direction.Crude, value: number): XY => {
  const xy = construct(c);
  if (direction === "x") return { x: value, y: xy.y };
  return { x: xy.x, y: value };
};

export const distance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.sqrt((a.x - b.x) ** 2 + (a.y - b.y) ** 2);
};

export const xDistance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.abs(a.x - b.x);
};

export const yDistance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.abs(a.y - b.y);
};

export const translation = (ca: Crude, cb: Crude): XY => {
  const a = construct(ca);
  const b = construct(cb);
  return { x: b.x - a.x, y: b.y - a.y };
};

export const isNan = (a: Crude): boolean => {
  const xy = construct(a);
  return Number.isNaN(xy.x) || Number.isNaN(xy.y);
};

export const couple = (a: Crude): NumberCouple => {
  const xy = construct(a);
  return [xy.x, xy.y];
};

export const css = (a: Crude): { left: number; top: number } => {
  const xy = construct(a);
  return { left: xy.x, top: xy.y };
};
