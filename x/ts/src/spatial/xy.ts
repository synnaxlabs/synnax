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
  xy,
  type NumberCouple,
  type ClientXY,
  type XY,
  signedDimensions,
  type Direction,
} from "@/spatial/base";

export { clientXY, xy, type ClientXY as Client, type XY };

/** A crude representation of a {@link XY} coordinate as a zod schema. */
export const crudeZ = z.union([
  xy,
  numberCouple,
  dimensions,
  signedDimensions,
  clientXY,
]);

/** A crude representation of a {@link XY} coordinate. */
export type Crude = z.infer<typeof crudeZ>;

/**
 * @constructs XY
 * @param x - A crude representation of the XY coordinate as a number, number couple,
 * dimensions, signed dimensions, or client XY.
 * @param y - If x is a number, the y coordinate. If x is a number and this argument is
 * not given, the y coordinate is assumed to be the same as the x coordinate.
 */
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

/** An x and y coordinate of NaN */
export const NAN = { x: NaN, y: NaN };

/** @returns true if the two XY coordinates are semntically equal. */
export const equals = (a: Crude, b: Crude): boolean => {
  const a_ = construct(a);
  const b_ = construct(b);
  return a_.x === b_.x && a_.y === b_.y;
};

/** Is zero is true if the XY coordinate has a semantic x and y value of zero. */
export const isZero = (c: Crude): boolean => equals(c, ZERO);

/**
 * @returns the given coordinate scaled by the given factors. If only one factor is given,
 * the y factor is assumed to be the same as the x factor.
 */
export const scale = (c: Crude, x: number, y: number = x): XY => {
  const p = construct(c);
  return { x: p.x * x, y: p.y * y };
};

/** @returns the given coordinate translated in the X direction by the given amount. */
export const translateX = (c: Crude, x: number): XY => {
  const p = construct(c);
  return { x: p.x + x, y: p.y };
};

/** @returns the given coordinate translated in the Y direction by the given amount. */
export const translateY = (c: Crude, y: number): XY => {
  const p = construct(c);
  return { x: p.x, y: p.y + y };
};

/**
 * @returns the given coordinate translated by an arbitrary number of translation
 * coordinates.
 */
export const translate = (a: Crude, b: Crude, ...cb: Crude[]): XY =>
  [a, b, ...cb].reduce((p: XY, c) => {
    const xy = construct(c);
    return { x: p.x + xy.x, y: p.y + xy.y };
  }, ZERO);

/**
 * @returns the given coordinate the given direction set to the given value.
 * @example set({ x: 1, y: 2 }, "x", 3) // { x: 3, y: 2 }
 */
export const set = (c: Crude, direction: Direction, value: number): XY => {
  const xy = construct(c);
  if (direction === "x") return { x: value, y: xy.y };
  return { x: xy.x, y: value };
};

/** @returns the magnitude of the distance between the two given coordinates. */
export const distance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.sqrt((a.x - b.x) ** 2 + (a.y - b.y) ** 2);
};

/** @returns the magnitude of the x distance between the two given coordinates. */
export const xDistance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.abs(a.x - b.x);
};

/** @returns the magnitude of the y distance between the two given coordinates. */
export const yDistance = (ca: Crude, cb: Crude): number => {
  const a = construct(ca);
  const b = construct(cb);
  return Math.abs(a.y - b.y);
};

/**
 * @returns the translation that would need to be applied to move the first coordinate
 * to the second coordinate.
 */
export const translation = (ca: Crude, cb: Crude): XY => {
  const a = construct(ca);
  const b = construct(cb);
  return { x: b.x - a.x, y: b.y - a.y };
};

/** @returns true if both the x and y coordinates of the given coordinate are NaN. */
export const isNan = (a: Crude): boolean => {
  const xy = construct(a);
  return Number.isNaN(xy.x) || Number.isNaN(xy.y);
};

/** @returns the coordinate represented as a couple of the form [x, y]. */
export const couple = (a: Crude): NumberCouple => {
  const xy = construct(a);
  return [xy.x, xy.y];
};

/** @returns the coordinate represented as css properties in the form { left, top }. */
export const css = (a: Crude): { left: number; top: number } => {
  const xy = construct(a);
  return { left: xy.x, top: xy.y };
};
