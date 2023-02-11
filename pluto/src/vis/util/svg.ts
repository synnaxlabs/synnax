import { Location, XY } from "@/spatial";

export const fTranslate = (v: XY): string => `translate(${v.x}, ${v.y})`;

export const fRotate = (v: number): string => `rotate(${v})`;

export const locationRotations: Record<Location, number> = {
  bottom: 180,
  top: 0,
  left: -90,
  right: 90,
  center: 0,
};
