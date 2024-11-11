// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const snakeToCamelStr = (str: string): string => {
  const c = str.replace(/_[a-z]/g, (m) => m[1].toUpperCase());
  // if both first and second characters are upper case, leave as is
  // if only first character is upper case, convert to lower case
  if (c.length > 1 && c[0] === c[0].toUpperCase() && c[1] === c[1].toUpperCase())
    return c;
  if (c.length === 0) return c;
  return c[0].toLowerCase() + c.slice(1);
};
/**
 * Convert string keys in an object to snake_case format.
 * @param obj: object to convert keys. If `obj` isn't a json object, `null` is returned.
 * @param opt: (optional) Options parameter, default is non-recursive.
 */
const createConverter = (
  f: (v: string) => string,
): (<V>(obj: V, opt?: Options) => V) => {
  const converter = <V>(obj: V, opt: Options = defaultOptions): V => {
    if (typeof obj === "string") return f(obj) as any;
    if (Array.isArray(obj)) return obj.map((v) => converter(v, opt)) as V;
    if (!isValidObject(obj)) return obj;
    opt = validateOptions(opt);
    const res: any = {};
    const anyObj = obj as any;
    Object.keys(anyObj).forEach((key) => {
      let value = anyObj[key];
      const nkey = f(key);
      if (opt.recursive)
        if (isValidObject(value)) {
          if (!belongToTypes(value, opt.keepTypesOnRecursion))
            value = converter(value, opt);
        } else if (opt.recursiveInArray && isArrayObject(value))
          value = [...value].map((v) => {
            let ret = v;
            if (isValidObject(v)) {
              // object in array
              if (!belongToTypes(ret, opt.keepTypesOnRecursion))
                ret = converter(v, opt);
            } else if (isArrayObject(v)) {
              // array in array
              // workaround by using an object holding array value
              const temp: any = converter({ key: v }, opt);
              ret = temp.key;
            }
            return ret;
          });

      res[nkey] = value;
    });

    return res;
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
  str.replace(/([a-z0-9])([A-Z])/g, (_, p1, p2) => `${p1}_${p2.toLowerCase()}`);
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
  recursive: boolean;
  recursiveInArray?: boolean;
  keepTypesOnRecursion?: any[];
}

/**
 * Default options for convert function. This option is not recursive.
 */
export const defaultOptions: Options = {
  recursive: true,
  recursiveInArray: true,
  keepTypesOnRecursion: [Number, String, Uint8Array],
};

export const validateOptions = (opt: Options = defaultOptions): Options => {
  if (opt.recursive == null) opt = defaultOptions;
  else opt.recursiveInArray ??= false;
  return opt;
};

export const isArrayObject = (obj: any): boolean => obj != null && Array.isArray(obj);

export const isValidObject = (obj: any): boolean =>
  obj != null && typeof obj === "object" && !Array.isArray(obj);

export const belongToTypes = (obj: any, types?: any[]): boolean =>
  (types || []).some((Type) => obj instanceof Type);
