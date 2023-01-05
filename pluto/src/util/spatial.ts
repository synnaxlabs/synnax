// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type Location = VerticalLocation | HorizontalLocation | CenterLocation;

export type VerticalLocation = "top" | "bottom";
export type HorizontalLocation = "left" | "right";
export type CenterLocation = "center";

export type Direction = "horizontal" | "vertical";

export type Position = "start" | "center" | "end";

export type Order = "first" | "last";

export const Locations = ["top", "bottom", "left", "right", "center"];

export const Directions = ["horizontal", "vertical"];

export const Positions = ["start", "center", "end"];

export const getDirection = (location: Location): Direction => {
  return location === "top" || location === "bottom" ? "horizontal" : "vertical";
};
export const swapDirection = (direction: Direction): Direction => {
  return direction === "horizontal" ? "vertical" : "horizontal";
};
export const getLocation = (direction: Direction): Location => {
  return direction === "horizontal" ? "left" : "top";
};
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
  { width, height }: { width: number; height: number }
): number => {
  return direction === "horizontal" ? width : height;
};

export interface Dimensions {
  width: number;
  height: number;
}
