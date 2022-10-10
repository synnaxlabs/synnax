export type Location = "top" | "bottom" | "left" | "right";
export type Direction = "horizontal" | "vertical";
export const getDirection = (location: Location): Direction => {
  return location === "top" || location === "bottom"
    ? "horizontal"
    : "vertical";
};
export const getLocation = (direction: Direction): Location => {
  return direction === "horizontal" ? "left" : "top";
};
