export type Location = VerticalLocation | HorizontalLocation | CenterLocation;

export type VerticalLocation = "top" | "bottom";
export type HorizontalLocation = "left" | "right";
export type CenterLocation = "center";

export type Direction = "horizontal" | "vertical";

export type Position = "start" | "center" | "end";

export type Order = "first" | "last";

export const Locations = ["top", "bottom", "left", "right", "center"];

export const Directions = ["horizontal", "vertical"];

export const isDirection = (v: string): boolean => Directions.includes(v as Direction);

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
  { width, height }: Dimensions
): number => {
  return direction === "horizontal" ? width : height;
};

export interface Dimensions {
  width: number;
  height: number;
}

/** A generic 2D point, scale, or offset. */
export interface XY {
  x: number;
  y: number;
}

export const ZERO_XY: XY = { x: 0, y: 0 };
export const ONE_XY: XY = { x: 1, y: 1 };
