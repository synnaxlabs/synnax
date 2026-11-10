// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";
import { type $ZodType } from "zod/v4/core";

/** What a field of an update schema changes, and the schemas it holds. */
export type Kind =
  | { kind: "value" }
  | { kind: "change"; value: z.ZodType }
  | { kind: "struct"; update: z.ZodType; full: z.ZodType }
  | { kind: "union"; update: z.ZodType }
  | { kind: "list"; entry: z.ZodType; update: z.ZodType }
  | { kind: "valueList"; entry: z.ZodType };

// Keys the kind that a field factory sets on the schema it returns.
const KIND = Symbol("patch.kind");

interface Marked {
  [KIND]?: Kind;
}

/** Sets the kind of a schema that a field factory returns, and returns the schema. */
export const mark = <S extends z.ZodType>(schema: S, kind: Kind): S =>
  Object.defineProperty(schema, KIND, { value: kind, enumerable: false });

const WRAPPERS = new Set([
  "optional",
  "default",
  "prefault",
  "nullable",
  "readonly",
  "lazy",
]);

/**
 * @returns the kind of the field schema, looking through the wrappers that a generated
 * update schema adds around a factory's schema, or undefined for a schema that no
 * factory returned.
 */
export const kindOf = (schema: $ZodType): Kind | undefined => {
  for (;;) {
    const kind = (schema as Marked)[KIND];
    if (kind != null) return kind;
    const def = schema._zod.def as {
      type: string;
      innerType?: $ZodType;
      getter?: () => $ZodType;
    };
    const inner = def.type === "lazy" ? def.getter?.() : def.innerType;
    if (!WRAPPERS.has(def.type) || inner == null) return undefined;
    schema = inner;
  }
};
