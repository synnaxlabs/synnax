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

/**
 * floatZ decodes a 64-bit float from a JSON number or the text "NaN", "Infinity", or
 * "-Infinity". z.encode writes a non-finite float back as its text, because JSON has
 * no non-finite number.
 */
export const floatZ = z.codec(
  z.union([z.number(), nonFiniteZ]),
  z.custom<number>((v) => typeof v === "number"),
  {
    decode: (v) => Number(v),
    encode: (v) => (Number.isFinite(v) ? v : nonFiniteZ.parse(String(v))),
  },
);

/** Float is a 64-bit float that may be NaN or infinite. */
export type Float = z.infer<typeof floatZ>;
