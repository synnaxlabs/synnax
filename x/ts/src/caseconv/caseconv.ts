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

/** Marks Zod schemas whose keys must not be converted. */
const PRESERVE_CASE_SYMBOL = "synnax.caseconv.preserveCase";

/**
 * Marks Zod record schemas whose keys are semantic values but whose values are
 * ordinary typed payloads: the keys stay untouched, the values still convert.
 */
const PRESERVE_KEYS_SYMBOL = "synnax.caseconv.preserveKeys";

interface ZodDef extends z.core.$ZodTypeDef {
  innerType?: z.core.SomeType;
  valueType?: z.core.SomeType;
  options?: readonly z.core.SomeType[];
  in?: z.core.SomeType;
  out?: z.core.SomeType;
  element?: z.core.SomeType;
  shape?: Record<string, z.ZodType>;
}

interface ZodInternals extends z.core.$ZodTypeInternals {
  def: ZodDef;
}

interface ZodSchema extends z.core.$ZodType {
  [PRESERVE_CASE_SYMBOL]?: boolean;
  [PRESERVE_KEYS_SYMBOL]?: boolean;
  _zod: ZodInternals;
  shape?: Record<string, z.ZodType>;
  sourceType?: () => { shape?: Record<string, z.ZodType> } | undefined;
}

/**
 * Marks a Zod schema so its keys and nested content skip case conversion. Use it where
 * keys are semantic values (OPC UA NodeIds, Modbus channel keys) rather than property
 * names.
 *
 * @example
 * const propertiesZ = z.object({
 *   read: z.object({
 *     channels: preserveCase(z.record(z.string(), z.number()))
 *   })
 * });
 */
export const preserveCase = <T extends z.ZodType>(schema: T): T => {
  (schema as ZodSchema)[PRESERVE_CASE_SYMBOL] = true;
  return schema;
};

/**
 * Marks a Zod record schema so its keys skip case conversion while the keys of its
 * values still convert. Use it for maps keyed by semantic identifiers (element keys,
 * channel names) whose values are ordinary typed payloads.
 */
export const preserveKeys = <T extends z.ZodType>(schema: T): T => {
  (schema as ZodSchema)[PRESERVE_KEYS_SYMBOL] = true;
  return schema;
};

const hasPreserveKeysMarker = (schema: unknown): boolean =>
  schema != null && typeof schema === "object" && PRESERVE_KEYS_SYMBOL in schema;

/**
 * Extracts the value schema from a Zod record schema, traversing wrappers
 * (optional, nullable, default, pipe) and unions to find the inner record.
 */
const getRecordValueSchema = (
  schema: z.ZodType | z.core.SomeType | undefined,
): z.ZodType | undefined => {
  if (schema == null) return undefined;
  const def = (schema as ZodSchema)._zod?.def;
  if (def == null) return undefined;
  if (def.type === "record" && def.valueType != null) return def.valueType as z.ZodType;
  if (def.innerType != null) return getRecordValueSchema(def.innerType);
  if (def.in != null) return getRecordValueSchema(def.in);
  if (Array.isArray(def.options))
    for (const option of def.options) {
      const result = getRecordValueSchema(option);
      if (result != null) return result;
    }
  return undefined;
};

/**
 * @returns true if the schema, or any schema it wraps (optional, nullable, union,
 * transform), carries the preserve-case marker.
 */
const hasPreserveCaseMarker = (schema: unknown): boolean => {
  if (schema == null || typeof schema !== "object") return false;

  if (PRESERVE_CASE_SYMBOL in schema) return true;

  const s = schema as ZodSchema;
  const def: ZodDef | undefined = s._zod?.def;
  if (def == null) return false;

  if (def.innerType && hasPreserveCaseMarker(def.innerType)) return true;

  if (def.type === "union" && Array.isArray(def.options))
    return def.options.some(hasPreserveCaseMarker);

  if (def.type === "pipe")
    return hasPreserveCaseMarker(def.in) || hasPreserveCaseMarker(def.out);

  return false;
};

/**
 * @returns the element schema of an array, looking through wrappers and unions, or
 * undefined when the schema is not an array.
 */
const getArrayElementSchema = (
  schema: z.ZodType | z.core.SomeType | undefined,
): z.ZodType | undefined => {
  if (schema == null) return undefined;
  const def = (schema as ZodSchema)._zod?.def;
  if (def?.type === "array" && def.element != null) return def.element as z.ZodType;
  if (def?.innerType != null) return getArrayElementSchema(def.innerType);
  if (def?.type === "union" && Array.isArray(def.options))
    for (const option of def.options) {
      const result = getArrayElementSchema(option);
      if (result != null) return result;
    }

  return undefined;
};

/**
 * @returns the shape (field name to ZodType) of an object schema, looking through
 * wrappers, or null for non-object schemas. Union options are merged with earlier
 * options winning key conflicts, so a lookup of one field name resolves to the variant
 * that declares it.
 */
const getSchemaShape = (
  schema: z.ZodType | z.core.SomeType | undefined,
): Record<string, z.ZodType> | null => {
  if (schema == null) return null;
  const s = schema as ZodSchema;
  if (s.shape != null) return s.shape;
  if (typeof s.sourceType === "function") {
    const st = s.sourceType();
    if (st?.shape != null) return st.shape;
  }
  const def = s._zod?.def;
  if (def == null) return null;
  if (def.innerType != null) return getSchemaShape(def.innerType);
  if (def.type === "union" && Array.isArray(def.options)) {
    let merged: Record<string, z.ZodType> | null = null;
    for (const option of def.options) {
      const result = getSchemaShape(option);
      if (result == null) continue;
      if (merged == null) merged = { ...result };
      else for (const k in result) if (!(k in merged)) merged[k] = result[k];
    }
    return merged;
  }

  if (def.type === "pipe") return getSchemaShape(def.in) ?? getSchemaShape(def.out);
  return null;
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

const createConverter = (
  f: (v: string) => string,
): (<V>(obj: V, opt?: Options) => V) => {
  const converter = <V>(obj: V, opt: Options = defaultOptions): V => {
    if (typeof obj === "string") return f(obj) as V;
    if (Array.isArray(obj)) {
      const elementSchema = getArrayElementSchema(opt.schema);
      const elemOpt: Options = {
        recursive: opt.recursive,
        recursiveInArray: opt.recursiveInArray,
        schema: elementSchema,
      };
      return obj.map((v) => converter(v, elemOpt)) as V;
    }
    if (!isValidObject(obj)) return obj;

    if (opt.schema != null && hasPreserveCaseMarker(opt.schema)) return obj;

    if (opt.schema != null && hasPreserveKeysMarker(opt.schema)) {
      const valueSchema = getRecordValueSchema(opt.schema);
      const childOpt: Options = {
        recursive: opt.recursive,
        recursiveInArray: opt.recursiveInArray,
        schema: valueSchema,
      };
      const res: record.Unknown = {};
      const anyObj = obj as record.Unknown;
      for (const key of Object.keys(anyObj)) {
        let value = anyObj[key];
        if (isValidObject(value)) {
          if (!isPreservedType(value)) value = converter(value, childOpt);
        } else if (Array.isArray(value)) {
          const elemOpt: Options = {
            recursive: opt.recursive,
            recursiveInArray: opt.recursiveInArray,
            schema: getArrayElementSchema(valueSchema),
          };
          value = (value as unknown[]).map((v) =>
            isValidObject(v) && !isPreservedType(v) ? converter(v, elemOpt) : v,
          );
        }
        res[key] = value;
      }
      return res as V;
    }

    const recursive = opt.recursive ?? true;
    const recursiveInArray = opt.recursiveInArray ?? recursive;
    const schema = opt.schema;
    const res: record.Unknown = {};
    const anyObj = obj as record.Unknown;
    if ("toJSON" in anyObj && typeof anyObj.toJSON === "function")
      return converter(anyObj.toJSON(), opt);

    const shape = getSchemaShape(schema);
    const childOpt: Options = { recursive, recursiveInArray, schema: undefined };
    const keys = Object.keys(anyObj);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      let value = anyObj[key];
      const nkey = f(key);

      // Look up schema using BOTH original key and converted key since:
      // - For snakeToCamel: schema has camelCase keys, input has snake_case, nkey is camelCase (matches)
      // - For camelToSnake: schema has camelCase keys, input has camelCase, nkey is snake_case (key matches)
      const propSchema: z.ZodType | undefined =
        shape != null ? (shape[key] ?? shape[nkey] ?? undefined) : undefined;

      if (recursive)
        if (isValidObject(value)) {
          if (!isPreservedType(value)) {
            childOpt.schema = propSchema;
            value = converter(value, childOpt);
          }
        } else if (recursiveInArray && Array.isArray(value)) {
          const elementSchema = getArrayElementSchema(propSchema);
          childOpt.schema = elementSchema;
          value = (value as unknown[]).map((v) => {
            if (isValidObject(v)) {
              if (!isPreservedType(v)) return converter(v, childOpt);
            } else if (Array.isArray(v)) {
              const temp: record.Unknown = converter({ key: v }, childOpt);
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
 */
export const snakeToCamel = createConverter(snakeToCamelStr);

const UPPER_A = "A".charCodeAt(0);
const UPPER_Z = "Z".charCodeAt(0);
const LOWER_A = "a".charCodeAt(0);
const LOWER_Z = "z".charCodeAt(0);
const DIGIT_0 = "0".charCodeAt(0);
const DIGIT_9 = "9".charCodeAt(0);
const UPPER_TO_LOWER = LOWER_A - UPPER_A;

// camelToSnakeStr inverts snakeToCamel. The first word break is always a capital
// following a lowercase or digit, so the prefix scan finds it without allocating
// (an already-snake string is returned as-is). From there a capital breaks a word
// if it follows a lowercase/digit or continues an uppercase run entered from one,
// so fooXY (from foo_x_y) splits fully while a leading run like NS=1;ID=5 stays put.
const camelToSnakeStr = (str: string): string => {
  const n = str.length;
  let i = 1;
  for (; i < n; i++) {
    const c = str.charCodeAt(i);
    if (c < UPPER_A || c > UPPER_Z) continue;
    const prev = str.charCodeAt(i - 1);
    if ((prev >= LOWER_A && prev <= LOWER_Z) || (prev >= DIGIT_0 && prev <= DIGIT_9))
      break;
  }
  if (i >= n) return str;
  let res = str.slice(0, i);
  // enteredFromLower stays true across an uppercase run begun after a lowercase or
  // digit, so every capital in fooXY splits rather than just the first.
  let enteredFromLower = true;
  for (; i < n; i++) {
    const c = str.charCodeAt(i);
    if (c < UPPER_A || c > UPPER_Z) {
      enteredFromLower = false;
      res += str[i];
      continue;
    }
    const prev = str.charCodeAt(i - 1);
    if ((prev >= LOWER_A && prev <= LOWER_Z) || (prev >= DIGIT_0 && prev <= DIGIT_9))
      enteredFromLower = true;
    else if (prev < UPPER_A || prev > UPPER_Z) enteredFromLower = false;
    res += enteredFromLower ? `_${String.fromCharCode(c + UPPER_TO_LOWER)}` : str[i];
  }
  return res;
};

/** Converts a camelCase string to snake_case. */
export const camelToSnake = createConverter(camelToSnakeStr);

/** Capitalizes the first character of the given string. */
export const capitalize = (str: string): string => {
  if (str.length === 0) return str;
  return str[0].toUpperCase() + str.slice(1);
};

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
  obj instanceof Uint8Array || obj instanceof Number || obj instanceof String;

/** Converts a string to kebab-case, splitting on spaces, camelCase, and capitals. */
const toKebabStr = (str: string): string =>
  str
    .replace(/[\s_]+/g, "-")
    .replace(
      /([a-z0-9])([A-Z])/g,
      (_, p1: string, p2: string) => `${p1}-${p2.toLowerCase()}`,
    )
    .toLowerCase();

/** Converts a string to kebab-case, splitting on spaces, camelCase, and capitals. */
export const toKebab = createConverter(toKebabStr);

/** Splits snake_case, kebab-case, camelCase, and PascalCase into spaced words. */
const splitWords = (str: string): string => {
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

  return result.replace(/\s+/g, " ").trim();
};

const toProperNounStr = (str: string): string => {
  if (str.length === 0) return str;
  return splitWords(str).replace(/\b\w/g, (char) => char.toUpperCase());
};

/** Converts a string to proper noun format, capitalizing every word. */
export const toProperNoun = createConverter(toProperNounStr);

// Matches acronym and version tokens ("XML", "V2") that keep their casing.
const ACRONYM_REGEX = /^[A-Z0-9]{2,}$/;

const toSentenceStr = (str: string): string => {
  if (str.length === 0) return str;
  const words = splitWords(str).split(" ");
  return words
    .map((word, i) => {
      if (ACRONYM_REGEX.test(word)) return word;
      const lower = word.toLowerCase();
      return i === 0 ? lower[0].toUpperCase() + lower.slice(1) : lower;
    })
    .join(" ");
};

/** Converts a string to sentence case. Acronyms keep their casing. */
export const toSentence = createConverter(toSentenceStr);
