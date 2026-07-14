// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** Validates an RFC 6901 JSON Pointer string (e.g. "/foo/bar/0"). */
export const pointerZ = z
  .string()
  .regex(/^(?:$|(?:\/(?:[^~/]|~0|~1)*)+)$/, "must be a valid JSON pointer (RFC 6901)");

/** A JSON primitive value: string, number, boolean, or null. */
export const primitiveZ = z.union([z.string(), z.number(), z.boolean(), z.null()]);

export type Primitive = z.infer<typeof primitiveZ>;

/** The type name of a JSON primitive: "string", "number", "boolean", or "null". */
export const primitiveTypeZ = z.enum(["string", "number", "boolean", "null"]);

export type PrimitiveType = z.infer<typeof primitiveTypeZ>;

export const ZERO_PRIMITIVES = {
  string: "",
  number: 0,
  boolean: false,
  null: null,
} as const satisfies Record<PrimitiveType, Primitive>;

const sortKeysDeep = (value: unknown): unknown => {
  if (Array.isArray(value)) return value.map(sortKeysDeep);
  if (value !== null && typeof value === "object") {
    const sorted: Record<string, unknown> = {};
    for (const key of Object.keys(value).sort())
      sorted[key] = sortKeysDeep((value as Record<string, unknown>)[key]);
    return sorted;
  }
  return value;
};

/**
 * Serializes value as canonical JSON: object keys sorted lexicographically and
 * compact separators. Matches the Go (x/go `json.Canonical`) and C++
 * (`x::json::canonical`) implementations, so the output hashes identically
 * across languages.
 */
export const canonicalString = (value: unknown): string =>
  JSON.stringify(sortKeysDeep(value));
