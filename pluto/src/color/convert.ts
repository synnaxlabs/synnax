import { RGBATuple } from "./types";

export const hexToRGBA = (hex: string, alpha: number = 1): RGBATuple => [
  p(hex, 1),
  p(hex, 3),
  p(hex, 5),
  alpha,
];

const p = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);
