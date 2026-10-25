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
import { narrow } from "@/narrow";

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

/** A discriminated union that a path goes into. */
export interface Discriminator {
  /** The path to the discriminator field of the union. */
  path: string[];
  /** The discriminator value that the document holds. */
  value: unknown;
}

/** Reads one path part off a document. A list entry is found by key, then by index. */
const child = (values: unknown, part: string): unknown => {
  if (Array.isArray(values)) {
    const entry = values.find((e: unknown) => narrow.isObject(e) && e.key === part);
    if (entry != null) return entry;
    const i = deep.getIndex(part);
    return i == null ? undefined : values[i];
  }
  return narrow.isObject(values) ? values[part] : undefined;
};

/**
 * Finds the schema at parts[index:] below schema, where values is the document at the
 * same place. Pushes each discriminated union that the path goes into onto found.
 */
const walk = (
  schema: $ZodType,
  parts: string[],
  index: number,
  values: unknown,
  found: Discriminator[],
): $ZodType | NotFound => {
  if (index >= parts.length) return schema;
  schema = unwrap(schema);
  const def = schema._zod.def as $ZodTypeDef & Record<string, unknown>;
  const part = parts[index];
  switch (def.type) {
    case "unknown":
    case "any":
      return schema;
    case "object": {
      const shape = def.shape as Record<string, $ZodType>;
      const best = deep.findBestKey(shape, parts.slice(index));
      if (best != null) {
        const [key, consumed] = best;
        return walk(shape[key], parts, index + consumed, child(values, key), found);
      }
      if (def.catchall == null) return NOT_FOUND;
      const catchall = def.catchall as $ZodType;
      return walk(catchall, parts, index + 1, child(values, part), found);
    }
    case "array":
      return walk(
        def.element as $ZodType,
        parts,
        index + 1,
        child(values, part),
        found,
      );
    case "tuple": {
      const i = deep.getIndex(part);
      if (i == null) return NOT_FOUND;
      const items = def.items as $ZodType[];
      const item = i < items.length ? items[i] : (def.rest as $ZodType | null);
      if (item == null) return NOT_FOUND;
      const value = Array.isArray(values) ? values[i] : undefined;
      return walk(item, parts, index + 1, value, found);
    }
    case "record":
    case "map":
      return walk(
        def.valueType as $ZodType,
        parts,
        index + 1,
        child(values, part),
        found,
      );
    case "union": {
      if (def.discriminator == null) {
        // A plain union is a migration or a loose alternative: the first member that
        // contains the path serves it.
        const length = found.length;
        for (const option of def.options as $ZodType[]) {
          const res = walk(option, parts, index, values, found);
          if (res !== NOT_FOUND) return res;
          found.length = length;
        }
        return NOT_FOUND;
      }
      const discriminator = def.discriminator as string;
      const value = narrow.isObject(values) ? values[discriminator] : undefined;
      const option = getDiscriminatedOption(
        schema as z.ZodDiscriminatedUnion,
        value as never,
      );
      if (option == null) return NOT_FOUND;
      found.push({ path: [...parts.slice(0, index), discriminator], value });
      return walk(option, parts, index, values, found);
    }
    default:
      return NOT_FOUND;
  }
};

/**
 * Finds the schema of the field at a dot-separated path.
 *
 * A segment may name an object key, the key or index of an array element, a tuple
 * index, or a record key. Wrappers such as optional, default, and pipe are transparent. Inside a
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
  const res = walk(schema, path.split(deep.SEPARATOR), 0, values, []);
  if (res !== NOT_FOUND) return res as z.ZodType;
  if (optional) return null;
  throw new Error(`Schema does not contain the path ${path}`);
}) as GetFieldSchema;

/**
 * Lists the discriminated unions that a path goes into, from the root down, with the
 * discriminator value that `values` holds for each. A union that the path ends at does
 * not count. A segment into an array names the `key` of an entry, or else its index. A
 * `z.unknown()` or `z.any()` holds no unions below it.
 *
 * @throws {Error} if the schema does not contain the path, or `values` holds no
 * discriminator for a union that the path goes into.
 */
export const getDiscriminators = (
  schema: z.ZodType,
  path: string[],
  values: unknown,
): Discriminator[] => {
  const found: Discriminator[] = [];
  if (walk(schema, path, 0, values, found) !== NOT_FOUND) return found;
  throw new Error(`Schema does not contain the path ${path.join(deep.SEPARATOR)}`);
};
