/**
 * RGBATuple represents a normalized RGBA color.
 */
export type RGBATuple = [number, number, number, number];

/**
 * hexToRGBATuple converts a hex color string to an RGBATuple.
 * @param hex - The hex color string to convert.
 * @param alpha - The alpha value to use for the color. Defaults to 1.
 * @returns A normalized RGBATuple representing the specified hex color.
 */
export const hexToRGBATuple = (hex: string, alpha = 1): RGBATuple => {
  let parseString = hex;
  if (hex.startsWith("#")) {
    parseString = hex.slice(1, 7);
  }
  if (parseString.length !== 6) {
    throw new Error(`[Pluto] - hexToRGB: invalid hex string: ${hex}`);
  }
  const r = parseInt(parseString.slice(0, 2), 16);
  const g = parseInt(parseString.slice(2, 4), 16);
  const b = parseInt(parseString.slice(4, 6), 16);
  if (isNaN(r) || isNaN(g) || isNaN(b)) {
    throw new Error(`[Pluto] - hexToRGB: invalid hex string: ${hex}`);
  }
  return [r / 255, g / 255, b / 255, alpha];
};
