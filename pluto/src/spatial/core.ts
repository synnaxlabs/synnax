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

export const Positions = ["start", "center", "end"];
export type Position = typeof Positions[number];

export const Orders = ["first", "last"] as const;
export type Order = typeof Orders[number];

export const VerticalLocations = ["top", "bottom"] as const;
export type VerticalLocation = typeof VerticalLocations[number];
export const HorizontalLocations = ["left", "right"] as const;
export type HorizontalLocation = typeof HorizontalLocations[number];
export type CenterLocation = "center";

export const Corners = ["topLeft", "topRight", "bottomLeft", "bottomRight"] as const;
export type Corner = typeof Corners[number];

export const Outerlocations = [...VerticalLocations, ...HorizontalLocations] as const;
export type OuterLocation = typeof Outerlocations[number];

export const Locations = [...Outerlocations, "center"] as const;
export type Location = typeof Locations[number];

export const Directions = ["horizontal", "vertical"] as const;
export type Direction = typeof Directions[number];
export const isDirection = (v: string): boolean => Directions.includes(v as Direction);

export const directionFromLocation = (location: Location): Direction =>
  VerticalLocations.includes(location as VerticalLocation) ? "horizontal" : "vertical";
export const swapDirection = (direction: Direction): Direction =>
  direction === "horizontal" ? "vertical" : "horizontal";
export const locationFromDirection = (direction: Direction): Location =>
  direction === "horizontal" ? "left" : "top";
export const swapLocation = (location: Location): Location => {
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
export const getDirectionalSize = (
  direction: Direction,
  { width, height }: Dimensions
): number => (direction === "horizontal" ? width : height);

export interface Dimensions {
  width: number;
  height: number;
}

/** A generic 2D point, scale, or offset. */
export interface XY {
  x: number;
  y: number;
}

export interface Transform {
  offset: XY;
  scale: XY;
}

export const ZERO_XY: XY = { x: 0, y: 0 };
export const ONE_XY: XY = { x: 1, y: 1 };
export const INFINITE_XY: XY = { x: Infinity, y: Infinity };
