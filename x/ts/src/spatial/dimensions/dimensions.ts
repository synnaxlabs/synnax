// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Dimensions,
  dimensionsZ,
  type NumberCouple,
  type SignedDimensions,
  type XY,
} from "@/spatial/base";

export { type Dimensions, dimensionsZ };

export type Crude = Dimensions | SignedDimensions | XY | NumberCouple;

export const ZERO: Dimensions = Object.freeze({ width: 0, height: 0 });
export const DECIMAL: Dimensions = Object.freeze({ width: 1, height: 1 });

export const construct = (width: number | Crude, height?: number): Dimensions => {
  if (typeof width === "number") return { width, height: height ?? width };
  if (Array.isArray(width)) return { width: width[0], height: width[1] };
  if ("x" in width) return { width: width.x, height: width.y };
  if ("signedWidth" in width)
    return { width: width.signedWidth, height: width.signedHeight };
  return { ...width };
};

export type Signed = SignedDimensions;

export const equals = (ca: Crude, cb?: Crude | null): boolean => {
  if (cb == null) return false;
  const a = construct(ca);
  const b = construct(cb);
  return a.width === b.width && a.height === b.height;
};

export const swap = (ca: Crude): Dimensions => {
  const a = construct(ca);
  return { width: a.height, height: a.width };
};

export const svgViewBox = (ca: Crude): string => {
  const a = construct(ca);
  return `0 0 ${a.width} ${a.height}`;
};

export const couple = (ca: Crude): [number, number] => {
  const a = construct(ca);
  return [a.width, a.height];
};

export const max = (crude: Crude[]): Dimensions => ({
  width: Math.max(...crude.map((c) => construct(c).width)),
  height: Math.max(...crude.map((c) => construct(c).height)),
});

export const scale = (ca: Crude, factor: number): Dimensions => {
  const a = construct(ca);
  return { width: a.width * factor, height: a.height * factor };
};
