// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const nonFiniteZ = z.enum(["NaN", "Infinity", "-Infinity"]);
const numberZ = z.custom<number>((v) => typeof v === "number");

/**
 * floatZ decodes a 64-bit float from a number or the text "NaN", "Infinity", or
 * "-Infinity". z.encode writes a non-finite float back as its text, because JSON has
 * no non-finite number.
 */
export const floatZ = z.codec(z.union([numberZ, nonFiniteZ]), numberZ, {
  decode: (v) => Number(v),
  encode: (v) => (Number.isFinite(v) ? v : nonFiniteZ.parse(String(v))),
});

/** Float is a 64-bit float that may be NaN or infinite. */
export type Float = z.infer<typeof floatZ>;

const decimalZ = (pattern: RegExp, min: bigint, max: bigint) =>
  z.string().refine((v) => {
    if (!pattern.test(v)) return false;
    const n = BigInt(v);
    return n >= min && n <= max;
  }, "must be decimal digits within range");

/** integerZ is a signed 64-bit integer as a string of decimal digits. */
export const integerZ = decimalZ(/^-?[0-9]+$/, -(2n ** 63n), 2n ** 63n - 1n);

/** Integer is a signed 64-bit integer as a string of decimal digits. */
export type Integer = z.infer<typeof integerZ>;

/** unsignedZ is an unsigned 64-bit integer as a string of decimal digits. */
export const unsignedZ = decimalZ(/^[0-9]+$/, 0n, 2n ** 64n - 1n);

/** Unsigned is an unsigned 64-bit integer as a string of decimal digits. */
export type Unsigned = z.infer<typeof unsignedZ>;
