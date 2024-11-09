// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const NOTATIONS = ["standard", "scientific", "engineering"] as const;
export const notationZ = z.enum(NOTATIONS);
export type Notation = z.infer<typeof notationZ>;

export const stringifyNumber = (
  value: number,
  precision: number,
  notation: Notation,
): string => {
  if (value === Infinity) return "∞";
  if (value === -Infinity) return "-∞";
  if (Number.isNaN(value)) return "NaN";
  if (notation === "standard") return value.toFixed(precision);
  if (notation === "scientific") return value.toExponential(precision);
  if (value === 0) return "0.00e0";
  const exp = Math.floor(Math.log10(Math.abs(value)) / 3) * 3;
  const mantissa = value / 10 ** exp;
  return `${mantissa.toFixed(precision)}e${exp}`;
};
