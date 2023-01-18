/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

export const POSITIONS = ["start", "center", "end"];
export type Position = typeof POSITIONS[number];

export const ORDERS = ["first", "last"] as const;
export type Order = typeof ORDERS[number];

export const Y_LOCATIONS = ["top", "bottom"] as const;
export type XLocation = typeof Y_LOCATIONS[number];
export const X_LOCATIONS = ["left", "right"] as const;
export type YLocation = typeof X_LOCATIONS[number];
export type CenterLocation = "center";

export const CORNERS = ["topLeft", "topRight", "bottomLeft", "bottomRight"] as const;
export type Corner = typeof CORNERS[number];

export const OUTER_LOCATIONS = [...Y_LOCATIONS, ...X_LOCATIONS] as const;
export type OuterLocation = typeof OUTER_LOCATIONS[number];

export const LOCATIONS = [...OUTER_LOCATIONS, "center"] as const;
export type Location = typeof LOCATIONS[number];

export const DIRECTIONS = ["x", "y"] as const;
export type Direction = typeof DIRECTIONS[number];
export const isDirection = (v: string): boolean => DIRECTIONS.includes(v as Direction);

export const locToDir = (location: Location | Direction): Direction => {
  if (isDirection(location)) return location as Direction;
  return Y_LOCATIONS.includes(location as XLocation) ? "y" : "x";
};

export const swapDir = (direction: Direction): Direction =>
  direction === "x" ? "y" : "x";

export const locFromDir = (direction: Direction): Location =>
  direction === "x" ? "left" : "top";

export const swapLoc = (location: Location): Location => {
  switch (location) {
    case "top":
      return "bottom";
    case "bottom":
      return "top";
    case "left":
      return "right";
    case "right":
      return "left";
    case "center":
      return "center";
  }
};

/** A generic 2D point, scale, or offset. */
export interface XY extends Record<Direction, number> {}

export const ZERO_XY: XY = { x: 0, y: 0 };
export const ONE_XY: XY = { x: 1, y: 1 };
export const INFINITE_XY: XY = { x: Infinity, y: Infinity };

export interface SignedDimensions {
  signedWidth: number;
  signedHeight: number;
}

export interface Dimensions {
  width: number;
  height: number;
}

export interface Transform {
  offset: XY;
  scale: XY;
}

export interface ClientXY {
  clientX: number;
  clientY: number;
}

export const toXY = (pt: XY | ClientXY | Dimensions): XY => {
  if ("x" in pt) return pt;
  if ("width" in pt) return { x: pt.width, y: pt.height };
  return { x: pt.clientX, y: pt.clientY };
};

export const locDim = (
  location: Location | Direction,
  point: XY | Dimensions
): number => toXY(point)[locToDir(location)];

export type ClientXYF = (e: ClientXY) => void;

export const dirToDim = (direction: Direction): "width" | "height" =>
  direction === "x" ? "width" : "height";
