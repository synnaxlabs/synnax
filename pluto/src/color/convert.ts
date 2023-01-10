import { RGBATuple } from "./types";

export const hexToRGBA = (hex: string, alpha: number = 1): RGBATuple => {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return [r, g, b, alpha];
};
