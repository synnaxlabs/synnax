// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import {
  type ClientXY,
  clientXY,
  dimensions,
  type Direction,
  type NumberCouple,
  numberCouple,
  signedDimensions,
  type XY,
  xy,
} from "@/spatial/base";

export { type ClientXY as Client, clientXY, type XY, xy };

/** A crude representation of a {@link XY} coordinate as a zod schema. */
export const crudeZ = z.union([
  z.number(),
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
 * dimensions, signed dimensions, or mouse event. If it's a mouse event, the clientX and
 * clientY coordinates are preferred over the x and y coordinates.
 * @param y - If x is a number, the y coordinate. If x is a number and this argument is
 * not given, the y coordinate is assumed to be the same as the x coordinate.
 */
export const construct = (x: Crude | Direction, y?: number): XY => {
  if (typeof x === "string") {
    if (y === undefined) throw new Error("The y coordinate must be given.");
    if (x === "x") return { x: y, y: 0 };
    return { x: 0, y };
  }
  // The order in which we execute these checks is very important.
  if (typeof x === "number") return { x, y: y ?? x };
  if (Array.isArray(x)) return { x: x[0], y: x[1] };
  if ("signedWidth" in x) return { x: x.signedWidth, y: x.signedHeight };
  if ("clientX" in x) return { x: x.clientX, y: x.clientY };
  if ("width" in x) return { x: x.width, y: x.height };
  return { x: x.x, y: x.y };
};

/** An x and y coordinate of zero */
export const ZERO: XY = Object.freeze({ x: 0, y: 0 });

/** An x and y coordinate of one */
export const ONE: XY = Object.freeze({ x: 1, y: 1 });

/** An x and y coordinate of infinity */
export const INFINITY: XY = Object.freeze({ x: Infinity, y: Infinity });

/** An x and y coordinate of NaN */
export const NAN: XY = Object.freeze({ x: NaN, y: NaN });

/** @returns true if the two XY coordinates are semantically equal. */
export const equals = (a: Crude, b: Crude, threshold: number = 0): boolean => {
  const a_ = construct(a);
  const b_ = construct(b);
  if (threshold === 0) return a_.x === b_.x && a_.y === b_.y;
  return Math.abs(a_.x - b_.x) <= threshold && Math.abs(a_.y - b_.y) <= threshold;
};

/** Is zero is true if the XY coordinate has a semantic x and y value of zero. */
export const isZero = (c: Crude): boolean => equals(c, ZERO);

/**
 * @returns the given coordinate scaled by the given factors. If only one factor is given,
 * the y factor is assumed to be the same as the x factor.
 */
export const scale = (c: Crude, x: number | Crude, y?: number): XY => {
  const p = construct(c);
  const xy = construct(x, y);
  return { x: p.x * xy.x, y: p.y * xy.y };
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

interface Translate {
  /** @returns the sum of the given coordinates. */
  (a: Crude, b: Crude, ...cb: Crude[]): XY;
  /** @returns the coordinates translated in the given direction by the given value. */
  (a: Crude, direction: Direction, value: number): XY;
}

export const translate: Translate = (a, b, v, ...cb): XY => {
  if (typeof b === "string" && typeof v === "number") {
    if (b === "x") return translateX(a, v);
    return translateY(a, v);
  }
  return [a, b, v ?? ZERO, ...cb].reduce((p: XY, c) => {
    const xy = construct(c as Crude);
    return { x: p.x + xy.x, y: p.y + xy.y };
  }, ZERO);
};

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

/**
 * @returns the translation that would need to be applied to move the first coordinate
 * to the second coordinate.
 */
export const translation = (to: Crude, from: Crude): XY => {
  const to_ = construct(to);
  const from_ = construct(from);
  return { x: from_.x - to_.x, y: from_.y - to_.y };
};

/** @returns true if both the x and y coordinates of the given coordinate are NaN. */
export const isNan = (a: Crude): boolean => {
  const xy = construct(a);
  return Number.isNaN(xy.x) || Number.isNaN(xy.y);
};

/** @returns true if both the x and y coordinates of the given coordinate are finite. */
export const isFinite = (a: Crude): boolean => {
  const xy = construct(a);
  return Number.isFinite(xy.x) && Number.isFinite(xy.y);
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

/**
 * Truncate the given coordinates to the given precision.
 * @param a - The coordinates to truncate.
 * @param precision - The number of decimal places to truncate to.
 * @returns The truncated coordinates.
 */
export const truncate = (a: Crude, precision: number = 0): XY => {
  const xy = construct(a);
  return { x: Number(xy.x.toFixed(precision)), y: Number(xy.y.toFixed(precision)) };
};

/**
 * Subtracts the second coordinate from the first coordinate.
 * @param a - The first coordinate.
 * @param b - The second coordinate.
 * @returns The difference between the two coordinates.
 */
export const sub = (a: Crude, b: Crude): XY => {
  const xy = construct(a);
  const xy_ = construct(b);
  return { x: xy.x - xy_.x, y: xy.y - xy_.y };
};

/**
 * Interprets the given coordinates as a vector and returns the normal of the given
 * vector.
 * @param a - The coordinates to get the normal of.
 * @returns The normal of the given coordinates.
 */
export const normal = (a: Crude): XY => {
  const xy = construct(a);
  const length = Math.hypot(xy.x, xy.y);
  if (length === 0) return { x: 0, y: 0 };
  return { x: -xy.y / length, y: xy.x / length };
};

/**
 * Interprets the given coordinates as a vector and returns the unit vector of the given
 * vector.
 * @param a - The coordinates to get the unit vector of.
 * @returns The unit vector of the given coordinates.
 */
export const normalize = (a: Crude): XY => {
  const xy = construct(a);
  const length = Math.hypot(xy.x, xy.y);
  if (length === 0) return { x: 0, y: 0 };
  return { x: xy.x / length, y: xy.y / length };
};

/**
 * @returns the average of the given coordinates.
 * @param coordinates - The coordinates to average.
 */
export const average = (...coordinates: Crude[]): XY => {
  const sum = coordinates.reduce((p, c) => translate(p, c), ZERO);
  return scale(sum, 1 / coordinates.length);
};

/**
 * Calculates the miter vectors for the given path and offset. This function is useful
 * for calculate the translations need to create an offset and parallel path to the
 * given path.
 * @param path - The path to calculate the miters for.
 * @param offset - The magnitude of the miter vectors.
 * @returns The miter vectors for the given path.
 */
export const calculateMiters = (path: XY[], offset: number): XY[] => {
  const miters: XY[] = [];
  for (let i = 0; i < path.length; i++) {
    const currPoint = path[i];
    let normalPrev: XY;
    let normalNext: XY;
    let miterNormal: XY;
    let miterLength: number;
    if (i === 0) {
      const nextPoint = path[i + 1];
      const dirNext = sub(nextPoint, currPoint);
      normalNext = normal(dirNext);
      miterNormal = normalNext;
      miterLength = offset;
    } else if (i === path.length - 1) {
      const prevPoint = path[i - 1];
      const dirPrev = sub(currPoint, prevPoint);
      normalPrev = normal(dirPrev);
      miterNormal = normalPrev;
      miterLength = offset;
    } else {
      const prevPoint = path[i - 1];
      const nextPoint = path[i + 1];
      const dirPrev = sub(currPoint, prevPoint);
      const dirNext = sub(nextPoint, currPoint);
      normalPrev = normal(dirPrev);
      normalNext = normal(dirNext);
      const angle = Math.acos(
        (dirPrev.x * dirNext.x + dirPrev.y * dirNext.y) /
          (Math.hypot(dirPrev.x, dirPrev.y) * Math.hypot(dirNext.x, dirNext.y)),
      );
      const sinHalfAngle = Math.sin(angle / 2);
      if (sinHalfAngle === 0) miterLength = offset;
      else miterLength = offset / sinHalfAngle;
      miterNormal = normalize(average(normalPrev, normalNext));
    }
    miters.push(scale(miterNormal, miterLength));
  }
  return miters;
};
