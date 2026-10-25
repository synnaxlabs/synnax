// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";
import {
  type $ZodFunction,
  type $ZodFunctionIn,
  type $ZodFunctionOut,
  type $ZodType,
  type $ZodTypeDef,
  getDiscriminatedOption,
} from "zod/v4/core";

import { deep } from "@/deep";

export const functionOutput = <
  In extends $ZodFunctionIn,
  Out extends $ZodFunctionOut,
  Func extends $ZodFunction<In, Out>,
>(
  schema: $ZodFunction<In, Out>,
): Func["_zod"]["def"]["output"] => schema._zod.def.output;

/** Options for {@link getFieldSchema}. */
export interface GetFieldSchemaOptions<
  O extends boolean | undefined = boolean | undefined,
> {
  /** Whether a path the schema does not contain returns null instead of throwing. */
  optional?: O;
  /**
   * The document the path is read against. Needed to resolve a path through a
   * discriminated union.
   */
  values?: unknown;
}

export interface GetFieldSchema {
  (schema: z.ZodType, path: string, options?: GetFieldSchemaOptions<false>): z.ZodType;
  (
    schema: z.ZodType,
    path: string,
    options?: GetFieldSchemaOptions<boolean | undefined>,
  ): z.ZodType | null;
}

const NOT_FOUND = Symbol("notFound");
type NotFound = typeof NOT_FOUND;

/** Resolves an optional, default, nullable, pipe, or lazy wrapper to what it wraps. */
export const unwrap = (schema: $ZodType): $ZodType => {
  for (;;) {
    const def = schema._zod.def as $ZodTypeDef & Record<string, unknown>;
    switch (def.type) {
      case "optional":
      case "default":
      case "prefault":
      case "nullable":
      case "readonly":
      case "catch":
      case "nonoptional":
        schema = def.innerType as $ZodType;
        break;
      case "pipe": {
        // A preprocess pipes a transform into the schema; a transform pipes out of it.
        const input = def.in as $ZodType;
        schema = input._zod.def.type === "transform" ? (def.out as $ZodType) : input;
        break;
      }
      case "lazy":
        schema = (def.getter as () => $ZodType)();
        break;
      default:
        return schema;
    }
  }
};

const walk = (
  schema: $ZodType,
  parts: string[],
  index: number,
  values: unknown,
): $ZodType | NotFound => {
  if (index >= parts.length) return schema;
  schema = unwrap(schema);
  const def = schema._zod.def as $ZodTypeDef & Record<string, unknown>;
  switch (def.type) {
    case "unknown":
    case "any":
      return schema;
    case "object": {
      const shape = def.shape as Record<string, $ZodType>;
      for (let i = parts.length - index; i >= 1; i--) {
        const key = parts.slice(index, index + i).join(deep.SEPARATOR);
        if (key in shape) return walk(shape[key], parts, index + i, values);
      }
      if (def.catchall != null)
        return walk(def.catchall as $ZodType, parts, index + 1, values);
      return NOT_FOUND;
    }
    case "array":
      return walk(def.element as $ZodType, parts, index + 1, values);
    case "tuple": {
      const i = deep.getIndex(parts[index]);
      if (i == null) return NOT_FOUND;
      const items = def.items as $ZodType[];
      const item = i < items.length ? items[i] : (def.rest as $ZodType | null);
      if (item == null) return NOT_FOUND;
      return walk(item, parts, index + 1, values);
    }
    case "record":
    case "map":
      return walk(def.valueType as $ZodType, parts, index + 1, values);
    case "union": {
      if (def.discriminator == null) {
        // A plain union is a migration or a loose alternative: the first member that
        // contains the path serves it.
        for (const option of def.options as $ZodType[]) {
          const res = walk(option, parts, index, values);
          if (res !== NOT_FOUND) return res;
        }
        return NOT_FOUND;
      }
      if (values == null) return NOT_FOUND;
      const prefix = parts.slice(0, index).join(deep.SEPARATOR);
      const current = deep.get(values, prefix, {
        optional: true,
      });
      if (current == null) return NOT_FOUND;
      const option = getDiscriminatedOption(
        schema as z.ZodDiscriminatedUnion,
        current[def.discriminator as string] as never,
      );
      if (option == null) return NOT_FOUND;
      return walk(option, parts, index, values);
    }
    default:
      return NOT_FOUND;
  }
};

/**
 * Finds the schema of the field at a dot-separated path.
 *
 * A segment may name an object key, an array index or element key, a tuple index, or
 * a record key. Wrappers such as optional, default, and pipe are transparent. Inside a
 * discriminated union the member is chosen by the discriminator `values` holds there;
 * inside a plain union, by the first member that contains the path. A `z.unknown()` or
 * `z.any()` accepts any remaining path.
 *
 * @throws {Error} if the schema does not contain the path and `optional` is not set. A
 * discriminated union with no discriminator in `values` contains no path.
 */
export const getFieldSchema: GetFieldSchema = ((
  schema: z.ZodType,
  path: string,
  { optional = false, values }: GetFieldSchemaOptions = {},
): z.ZodType | null => {
  if (path === "") return schema;
  const res = walk(schema, path.split(deep.SEPARATOR), 0, values);
  if (res !== NOT_FOUND) return res as z.ZodType;
  if (optional) return null;
  throw new Error(`Schema does not contain the path ${path}`);
}) as GetFieldSchema;
