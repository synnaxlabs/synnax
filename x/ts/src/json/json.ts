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
