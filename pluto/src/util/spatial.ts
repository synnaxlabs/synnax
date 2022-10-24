export type Location = "top" | "bottom" | "left" | "right" | "center";

export type Direction = "horizontal" | "vertical";

export type Position = "start" | "center" | "end";

export type Order = "first" | "last";

export const Locations = ["top", "bottom", "left", "right"];

export const Directions = ["horizontal", "vertical"];

export const Positions = ["start", "center", "end"];

export const getDirection = (location: Location): Direction => {
  return location === "top" || location === "bottom"
    ? "horizontal"
    : "vertical";
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
