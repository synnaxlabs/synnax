// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { math } from "@/math";

export const int8Z = z.int().min(math.MIN_INT8).max(math.MAX_INT8);
export const int16Z = z.int().min(math.MIN_INT16).max(math.MAX_INT16);
export const int64Z = z.int().min(math.MIN_INT64_NUMBER).max(math.MAX_INT64_NUMBER);

export const uint8Z = z.int().min(0).max(math.MAX_UINT8);
export const uint12Z = z.int().min(0).max(math.MAX_UINT12);
export const uint16Z = z.int().min(0).max(math.MAX_UINT16);
export const uint20Z = z.int().min(0).max(math.MAX_UINT20);

// JSON utilities
const defaultJSONSchema = z.record(z.string(), z.unknown());

/**
 * Creates a schema that parses a JSON string and validates against an optional inner schema.
 * @param schema - Optional schema to validate the parsed JSON against. Defaults to record.
 * @returns A Zod schema that parses JSON strings.
 */
export const stringifiedJSON = <T extends z.ZodType = typeof defaultJSONSchema>(
  schema?: T,
) => {
  const inner = (schema ?? defaultJSONSchema) as T;
  return z
    .union([z.string(), z.record(z.string(), z.unknown())])
    .transform((s) => (typeof s === "string" ? JSON.parse(s) : s))
    .pipe(inner);
};

/**
 * Creates a schema that validates against an optional inner schema and stringifies to JSON.
 * @param schema - Optional schema to validate input against. Defaults to record.
 * @returns A Zod schema that stringifies values to JSON.
 */
export const jsonStringifier = <T extends z.ZodType = typeof defaultJSONSchema>(
  schema?: T,
) => {
  const inner = (schema ?? defaultJSONSchema) as T;
  return inner.transform((v) => JSON.stringify(v));
};

/** @deprecated Use uint12Z instead */
export const uint12 = uint12Z;
