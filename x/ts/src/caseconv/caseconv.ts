// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type record } from "@/record";
import { zod as zodUtil } from "@/zod";

/**
 * Global symbol used to mark Zod schemas that should not have their keys converted.
 * Uses Symbol.for() to ensure the same symbol is used across different module instances.
 */
const PRESERVE_CASE_SYMBOL = "synnax.caseconv.preserveCase";

/**
 * Marks a Zod schema to prevent case conversion of its keys and nested content.
 * Use this for schemas where keys are semantic values (like OPC UA NodeIds or Modbus channel keys)
 * rather than property names.
 *
 * @param schema - The Zod schema to mark
 * @returns The same schema with a preserve case marker
 *
 * @example
 * const propertiesZ = z.object({
 *   read: z.object({
 *     channels: preserveCase(z.record(z.string(), z.number()))
 *   })
 * });
 */
export const preserveCase = <T extends z.ZodType>(schema: T): T => {
  (schema as any)[PRESERVE_CASE_SYMBOL] = true;
  return schema;
};

/**
 * Checks if a Zod schema has the preserve case marker.
 * Traverses through wrapper schemas (optional, nullable, union, transform, etc.)
 * to find markers on inner schemas.
 */
const hasPreserveCaseMarker = (schema: unknown): boolean => {
  if (schema == null || typeof schema !== "object") return false;

  // Direct marker check
  if (PRESERVE_CASE_SYMBOL in schema) return true;

  const def = (schema as any)._zod?.def ?? (schema as any).def;
  if (def == null) return false;

  // Traverse through wrappers with innerType (optional, nullable, default, catch)
  if (def.innerType && hasPreserveCaseMarker(def.innerType)) return true;

  // Traverse through unions - check all options
  if (def.type === "union" && Array.isArray(def.options))
    return def.options.some(hasPreserveCaseMarker);

  // Traverse through pipes/transforms - check both ends
  if (def.type === "pipe")
    return hasPreserveCaseMarker(def.in) || hasPreserveCaseMarker(def.out);

  return false;
};

/**
 * Unwraps an array schema to get its element schema.
 * Handles direct arrays and unions containing arrays (e.g., from nullishToEmpty).
 * Returns undefined if the schema is not an array or is undefined.
 */
const getArrayElementSchema = (
  schema: z.ZodType | undefined,
): z.ZodType | undefined => {
  if (schema == null) return undefined;
  const def = (schema as any).def;
  if (def?.type === "array" && def.element != null) return def.element;
  // Handle union types that may contain arrays (e.g., nullishToEmpty)
  if (def?.type === "union" && Array.isArray(def.options))
    for (const option of def.options) {
      const result = getArrayElementSchema(option);
      if (result != null) return result;
    }

  return undefined;
};

const snakeToCamelStr = (str: string): string => {
  if (str.length === 0) return str;
  const hasUnderscore = str.indexOf("_") !== -1;
  const c = hasUnderscore ? str.replace(/_[a-z]/g, (m) => m[1].toUpperCase()) : str;
  const first = c.charCodeAt(0);
  if (first < 65 || first > 90) return c; // not uppercase A-Z
  if (c.length > 1 && c.charCodeAt(1) >= 65 && c.charCodeAt(1) <= 90) return c;
  return String.fromCharCode(first + 32) + c.slice(1);
};

/**
 * Convert string keys in an object to snake_case format.
 * @param obj: object to convert keys. If `obj` isn't a json object, `null` is returned.
 * @param opt: (optional) Options parameter, default is non-recursive.
 * @param schema: (optional) Zod schema to check for preserve case markers
 */
const createConverter = (
  f: (v: string) => string,
): (<V>(obj: V, opt?: Options) => V) => {
  const converter = <V>(obj: V, opt: Options = defaultOptions): V => {
    if (typeof obj === "string") return f(obj) as any;
    if (Array.isArray(obj)) {
      const elementSchema = getArrayElementSchema(opt.schema);
      return obj.map((v) => converter(v, { ...opt, schema: elementSchema })) as V;
    }
    if (!isValidObject(obj)) return obj;

    if (opt.schema != null && hasPreserveCaseMarker(opt.schema)) return obj;

    const recursive = opt.recursive ?? true;
    const recursiveInArray = opt.recursiveInArray ?? recursive;
    const schema = opt.schema;
    const res: record.Unknown = {};
    const anyObj = obj as record.Unknown;
    if ("toJSON" in anyObj && typeof anyObj.toJSON === "function")
      return converter(anyObj.toJSON(), opt);

    const keys = Object.keys(anyObj);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      let value = anyObj[key];
      const nkey = f(key);

      // Look up schema using BOTH original key and converted key since:
      // - For snakeToCamel: schema has camelCase keys, input has snake_case, nkey is camelCase (matches)
      // - For camelToSnake: schema has camelCase keys, input has camelCase, nkey is snake_case (key matches)
      const propSchema: z.ZodType | undefined =
        schema != null
          ? (zodUtil.getFieldSchema(schema, key, { optional: true }) ??
             zodUtil.getFieldSchema(schema, nkey, { optional: true }) ??
             undefined)
          : undefined;

      if (recursive)
        if (isValidObject(value)) {
          if (!isPreservedType(value))
            value = converter(value, {
              recursive,
              recursiveInArray,
              schema: propSchema,
            });
        } else if (recursiveInArray && Array.isArray(value)) {
          const elementSchema = getArrayElementSchema(propSchema);
          value = (value as unknown[]).map((v) => {
            if (isValidObject(v)) {
              if (!isPreservedType(v))
                return converter(v, {
                  recursive,
                  recursiveInArray,
                  schema: elementSchema,
                });
            } else if (Array.isArray(v)) {
              const temp: record.Unknown = converter(
                { key: v },
                { recursive, recursiveInArray, schema: elementSchema },
              );
              return temp.key;
            }
            return v;
          });
        }

      res[nkey] = value;
    }

    return res as V;
  };
  return converter;
};

/**
 * SnakeToCamel converts the given value from snake_case to camelCase. Note that this
 * function will ONLY convert snake_case, not any other case. For example, a value
 * like "foo-bar" will not be converted to "foo_bar". It will also not alter the
 * capitalization of the first character.
 *
 * @param value - A string, object, array of objects, or array of strings whose case
 * needs to be converted.
 * @returns A copy of the value with the case converted.
 */
export const snakeToCamel = createConverter(snakeToCamelStr);

const camelToSnakeStr = (str: string): string =>
  // Don't convert the first character and don't convert a character that is after a
  // non-alphanumeric character
  str.replace(
    /([a-z0-9])([A-Z])/g,
    (_, p1: string, p2: string) => `${p1}_${p2.toLowerCase()}`,
  );

/**
 * Converts a camelCase string to snake_case.
 *
 * @param str - The string to convert
 * @returns The converted string in snake_case
 */
export const camelToSnake = createConverter(camelToSnakeStr);

/**
 * Capitalize capitalizes the first character of the given string.
 *
 * @param str - The string to capitalize.
 * @returns The string with the first character capitalized.
 */
export const capitalize = (str: string): string => {
  if (str.length === 0) return str;
  return str[0].toUpperCase() + str.slice(1);
};

/**
 * Options parameter for convert function
 *
 * @param recursive: recursive if value of subkey is an object that is not an array
 * @param recursiveInArray: recursive if ${recursive} is `true` and value of subkey
 * is an array. All elements in array (value of subkey) will be recursive.
 * If ${recursiveInArray} is not set, default is `false`.
 * @param keepTypesOnRecursion: list of types will be keep value on recursion.
 * Example Date, RegExp. These types will be right-hand side of 'instanceof' operator.
 */
export interface Options {
  recursive?: boolean;
  recursiveInArray?: boolean;
  schema?: z.ZodType;
}

const defaultOptions: Options = {
  recursive: true,
  recursiveInArray: true,
  schema: undefined,
};

const isValidObject = (obj: unknown): boolean =>
  obj != null && typeof obj === "object" && !Array.isArray(obj);

const isPreservedType = (obj: unknown): boolean =>
  obj instanceof Number || obj instanceof String || obj instanceof Uint8Array;

/**
 * Converts a string to kebab-case.
 * Handles spaces, camelCase, and uppercase characters.
 *
 * @param str - The string to convert
 * @returns The converted string in kebab-case
 */
const toKebabStr = (str: string): string =>
  str
    .replace(/[\s_]+/g, "-")
    .replace(
      /([a-z0-9])([A-Z])/g,
      (_, p1: string, p2: string) => `${p1}-${p2.toLowerCase()}`,
    )
    .toLowerCase();

/**
 * Converts a string to kebab-case.
 * Handles spaces, camelCase, and uppercase characters.
 *
 * @param str - The string to convert
 * @returns The converted string in kebab-case
 */
export const toKebab = createConverter(toKebabStr);

/**
 * Converts a string to proper noun format.
 * Handles snake_case, kebab-case, camelCase, and PascalCase.
 * Capitalizes the first letter of each word.
 *
 * @param str - The string to convert
 * @returns The converted string in proper noun format
 */
const toProperNounStr = (str: string): string => {
  if (str.length === 0) return str;

  // Replace underscores and hyphens with spaces
  let result = str.replace(/[_-]/g, " ");

  // Insert spaces before capital letters (for camelCase/PascalCase)
  // but not at the start or when there are consecutive capitals
  result = result.replace(
    /([a-z0-9])([A-Z])/g,
    (_, p1: string, p2: string) => `${p1} ${p2}`,
  );

  // Handle consecutive capitals (e.g., "XMLParser" -> "XML Parser")
  result = result.replace(
    /([A-Z]+)([A-Z][a-z])/g,
    (_, p1: string, p2: string) => `${p1} ${p2}`,
  );

  // Clean up multiple spaces
  result = result.replace(/\s+/g, " ").trim();

  // Capitalize first letter of each word (proper noun format)
  result = result.replace(/\b\w/g, (char) => char.toUpperCase());

  return result;
};

/**
 * Converts a string to proper noun format.
 * Handles snake_case, kebab-case, camelCase, and PascalCase.
 * Each word is capitalized.
 *
 * @param str - The string to convert
 * @returns The converted string in proper noun format
 */
export const toProperNoun = createConverter(toProperNounStr);
