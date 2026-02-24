// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const colorMapZ = z.enum([
  "viridis",
  "inferno",
  "magma",
  "plasma",
  "jet",
  "grayscale",
]);
export type ColorMap = z.infer<typeof colorMapZ>;

type ColorStop = [number, number, number];

const interpolateStops = (
  stops: Array<[number, ColorStop]>,
): Uint8ClampedArray => {
  const lut = new Uint8ClampedArray(256 * 4);
  for (let i = 0; i < 256; i++) {
    const t = i / 255;
    let lo = 0;
    for (let j = 1; j < stops.length; j++) {
      if (stops[j][0] >= t) {
        lo = j - 1;
        break;
      }
      lo = j - 1;
    }
    const hi = Math.min(lo + 1, stops.length - 1);
    const range = stops[hi][0] - stops[lo][0];
    const f = range === 0 ? 0 : (t - stops[lo][0]) / range;
    const [, cLo] = stops[lo];
    const [, cHi] = stops[hi];
    const idx = i * 4;
    lut[idx] = Math.round(cLo[0] + (cHi[0] - cLo[0]) * f);
    lut[idx + 1] = Math.round(cLo[1] + (cHi[1] - cLo[1]) * f);
    lut[idx + 2] = Math.round(cLo[2] + (cHi[2] - cLo[2]) * f);
    lut[idx + 3] = 255;
  }
  return lut;
};

const VIRIDIS_STOPS: Array<[number, ColorStop]> = [
  [0.0, [68, 1, 84]],
  [0.25, [59, 82, 139]],
  [0.5, [33, 145, 140]],
  [0.75, [94, 201, 98]],
  [1.0, [253, 231, 37]],
];

const INFERNO_STOPS: Array<[number, ColorStop]> = [
  [0.0, [0, 0, 4]],
  [0.25, [87, 16, 110]],
  [0.5, [188, 55, 84]],
  [0.75, [249, 142, 9]],
  [1.0, [252, 255, 164]],
];

const MAGMA_STOPS: Array<[number, ColorStop]> = [
  [0.0, [0, 0, 4]],
  [0.25, [81, 18, 124]],
  [0.5, [183, 55, 121]],
  [0.75, [254, 136, 76]],
  [1.0, [252, 253, 191]],
];

const PLASMA_STOPS: Array<[number, ColorStop]> = [
  [0.0, [13, 8, 135]],
  [0.25, [126, 3, 168]],
  [0.5, [204, 71, 120]],
  [0.75, [248, 149, 64]],
  [1.0, [240, 249, 33]],
];

const JET_STOPS: Array<[number, ColorStop]> = [
  [0.0, [0, 0, 131]],
  [0.125, [0, 0, 255]],
  [0.375, [0, 255, 255]],
  [0.625, [255, 255, 0]],
  [0.875, [255, 0, 0]],
  [1.0, [128, 0, 0]],
];

const GRAYSCALE_STOPS: Array<[number, ColorStop]> = [
  [0.0, [0, 0, 0]],
  [1.0, [255, 255, 255]],
];

const buildLUT = (stops: Array<[number, ColorStop]>): Uint8ClampedArray =>
  interpolateStops(stops);

const LUTS: Record<ColorMap, Uint8ClampedArray> = {
  viridis: buildLUT(VIRIDIS_STOPS),
  inferno: buildLUT(INFERNO_STOPS),
  magma: buildLUT(MAGMA_STOPS),
  plasma: buildLUT(PLASMA_STOPS),
  jet: buildLUT(JET_STOPS),
  grayscale: buildLUT(GRAYSCALE_STOPS),
};

export const getLUT = (map: ColorMap): Uint8ClampedArray => LUTS[map];

export const mapValueToColor = (
  normalized: number,
  map: ColorMap,
  target: Uint8ClampedArray,
  offset: number,
): void => {
  const lut = LUTS[map];
  const idx = Math.max(0, Math.min(255, Math.round(normalized * 255))) * 4;
  target[offset] = lut[idx];
  target[offset + 1] = lut[idx + 1];
  target[offset + 2] = lut[idx + 2];
  target[offset + 3] = lut[idx + 3];
};
