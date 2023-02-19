// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RGBATuple } from "./core";

const invalidHexError = (hex: string): Error => new Error(`Invalid hex color: ${hex}`);

/**
 * Converts a hex color to an RGBA tuple. If the hex color is 8 characters long, the
 * alpha value will be set to the 9th character. Otherwise, the alpha value will be
 *  set to the value passed in.
 *
 * @param hex - The hex color to convert. Must be be between 6 and 8 characters long
 * (excluding the `#`).
 */
export const hexToRGBA = (
  hex: string,
  alpha: number = 1,
  normalize: number = 1
): RGBATuple => {
  if (!validateHex(hex)) throw invalidHexError(hex);
  hex = stripHash(hex);
  return normalizeRGBA(
    [p(hex, 0), p(hex, 2), p(hex, 4), hex.length === 9 ? p(hex, 7) / 255 : alpha],
    normalize
  );
};

/**
 * Validates a hex color with 6 or 8 characters and with or without a leading `#`.
 * @param hex - The hex color to validate.
 * @returns `true` if the hex color is valid, `false` otherwise.
 */
export const validateHex = (hex: string): boolean =>
  /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i.test(hex);

/**
 * Validates a hex color with 6 characters and with or without a leading `#`.
 * @param hex - The hex color to validate.
 * @returns `true` if the hex color is valid, `false` otherwise.
 */
export const validateSixCharHex = (hex: string): boolean =>
  /^#?([0-9a-f]{6})$/i.test(hex);

const stripHash = (hex: string): string => (hex.startsWith("#") ? hex.slice(1) : hex);

const p = (s: string, n: number): number => parseInt(s.slice(n, n + 2), 16);

/**
 * Normalizes an 8 bit RGBA tuple to values between 0 and 1.
 * @param t - The RGBA tuple to normalize.
 * @returns The normalized RGBA tuple.
 */
export const normalizeRGBA = (t: RGBATuple, divisor: number): RGBATuple => [
  t[0] / divisor,
  t[1] / divisor,
  t[2] / divisor,
  t[3],
];

/**
 * Adds an opacity to a hex color. If the hex color already has an opacity, the
 * opacity will be replaced.
 *
 * @param hex - The hex color to add an opacity to. Must be be between 6 and 8 characters long
 *   (excluding the `#`).
 * @param opacity - The opacity to add to the hex color. Must be between 0 and 100.
 * @returns The hex color with the opacity added.
 */
export const addOpacityToHex = (hex: string, opacity: Opacity): string => {
  hex = stripHash(hex);
  if (!validateSixCharHex(hex)) {
    hex = hex.slice(0, 6);
  }
  return `#${hex}${hexOpacities[opacity]}`;
};

const hexOpacities = {
  100: "FF",
  99: "FC",
  98: "FA",
  97: "F7",
  96: "F5",
  95: "F2",
  94: "F0",
  93: "ED",
  92: "EB",
  91: "E8",
  90: "E6",
  89: "E3",
  88: "E0",
  87: "DE",
  86: "DB",
  85: "D9",
  84: "D6",
  83: "D4",
  82: "D1",
  81: "CF",
  80: "CC",
  79: "C9",
  78: "C7",
  77: "C4",
  76: "C2",
  75: "BF",
  74: "BD",
  73: "BA",
  72: "B8",
  71: "B5",
  70: "B3",
  69: "B0",
  68: "AD",
  67: "AB",
  66: "A8",
  65: "A6",
  64: "A3",
  63: "A1",
  62: "9E",
  61: "9C",
  60: "99",
  59: "96",
  58: "94",
  57: "91",
  56: "8F",
  55: "8C",
  54: "8A",
  53: "87",
  52: "85",
  51: "82",
  50: "80",
  49: "7D",
  48: "7A",
  47: "78",
  46: "75",
  45: "73",
  44: "70",
  43: "6E",
  42: "6B",
  41: "69",
  40: "66",
  39: "63",
  38: "61",
  37: "5E",
  36: "5C",
  35: "59",
  34: "57",
  33: "54",
  32: "52",
  31: "4F",
  30: "4D",
  29: "4A",
  28: "47",
  27: "45",
  26: "42",
  25: "40",
  24: "3D",
  23: "3B",
  22: "38",
  21: "36",
  20: "33",
  19: "30",
  18: "2E",
  17: "2B",
  16: "29",
  15: "26",
  14: "24",
  13: "21",
  12: "1F",
  11: "1C",
  10: "1A",
  9: "17",
  8: "14",
  7: "12",
  6: "0F",
  5: "0D",
  4: "0A",
  3: "08",
  2: "05",
  1: "03",
  0: "00",
};

export type Opacity = keyof typeof hexOpacities;
