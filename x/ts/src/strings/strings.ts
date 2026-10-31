// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Joins an array of strings into a natural language list.
 *
 * @param strings - The array of strings to join.
 * @param zeroLength - The string to return if the array is empty. Defaults to an empty string.
 * @returns A string that represents the natural language list.
 *
 * @example
 * ```typescript
 * naturalLanguageJoin([]); // ""
 * naturalLanguageJoin([], "No items"); // "No items"
 * naturalLanguageJoin(["apple"]); // "apple"
 * naturalLanguageJoin(["apple", "banana"]); // "apple and banana"
 * naturalLanguageJoin(["apple", "banana", "cherry"]); // "apple, banana, and cherry"
 * ```
 */
export const naturalLanguageJoin = (
  strings: string | string[],
  zeroLength: string = "",
): string => {
  if (typeof strings === "string") return strings;
  const length = strings.length;
  if (length === 0) return zeroLength;
  if (length === 1) return strings[0];
  if (length === 2) return `${strings[0]} and ${strings[1]}`;
  return `${strings.slice(0, -1).join(", ")}, and ${strings[length - 1]}`;
};

/**
 * Creates a list of short identifiers from a given name.
 *
 * @param name - The name to create identifiers from.
 * @returns An array of unique short identifiers.
 *
 * @example
 * ```typescript
 * createShortIdentifiers("John Doe"); // ["jd", "j_d", "johdoe", "joh_doe"]
 * createShortIdentifiers("Alice 123"); // ["a1", "a_1", "a123", "a_12_3", "ali123", "ali_123"]
 * createShortIdentifiers("Bob"); // ["bob"]
 * ```
 */
export const createShortIdentifiers = (name: string): string[] => {
  const words = name.split(" ");
  const identifiers = new Set<string>();

  const initials = words.map((word) => word.charAt(0).toLowerCase()).join("");
  identifiers.add(initials.replace(/-/g, "_"));
  identifiers.add(initials.replace(/(.)(.)/g, "$1_$2").replace(/-/g, "_")); // Insert underscores

  const regex = /\d+/g;
  const hasNumbers = name.match(regex);

  if (hasNumbers)
    words.forEach((word, index) => {
      if (regex.test(word)) {
        const abbreviatedWords = words
          .map((w, i) => (i !== index ? w.charAt(0).toLowerCase() : w))
          .join("");
        identifiers.add(abbreviatedWords.replace(/-/g, "_"));
        identifiers.add(
          abbreviatedWords.replace(/(.)(.)/g, "$1_$2").replace(/-/g, "_"),
        ); // Insert underscores
      }
    });

  const wordAbbreviations = words.map((word) =>
    (word.length > 3 ? word.substring(0, 3) : word).toLowerCase(),
  );
  identifiers.add(wordAbbreviations.join("").replace(/-/g, "_"));
  identifiers.add(wordAbbreviations.join("_").replace(/-/g, "_"));

  const filteredIdentifiers = Array.from(identifiers).filter(
    (id) => id.length >= 2 && id.length <= 12 && !/^\d/.test(id),
  );

  return filteredIdentifiers;
};

/**
 * Removes a prefix from a string if it exists.
 *
 * @param str - The string to remove the prefix from.
 * @param prefix - The prefix to remove.
 * @returns The string with the prefix removed if it was present, otherwise the original string.
 *
 * @example
 * ```typescript
 * trimPrefix("hello world", "hello "); // "world"
 * trimPrefix("hello world", "goodbye "); // "hello world"
 * trimPrefix("hello world", ""); // "hello world"
 * trimPrefix("hello world", "hello world"); // ""
 * ```
 */
export const trimPrefix = (str: string, prefix: string): string => {
  if (str.startsWith(prefix)) return str.slice(prefix.length);
  return str;
};

/**
 * Escapes HTML special characters in a string to prevent XSS and ensure
 * correct rendering in HTML contexts.
 *
 * @param s - The string to escape.
 * @returns The escaped string with &, <, >, ", and ' replaced by their
 * HTML entity equivalents.
 */
export const escapeHTML = (s: string): string =>
  s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

/**
 * Formats a float as the shortest text that parses back to it at the given width. The
 * text is in exponent notation below 1e-6 or from 1e21 (1e-7, 1e+21). NaN is NaN, and
 * infinities are Infinity and -Infinity. It matches the float text of every Synnax
 * runtime.
 *
 * @param value - The float to format.
 * @param bits - The width of the float, 32 or 64.
 * @returns The text of value.
 */
export const formatFloat = (value: number, bits: 32 | 64 = 64): string => {
  const f32 = Math.fround(value);
  if (Object.is(value, -0) || (bits === 32 && Object.is(f32, -0))) return "-0";
  if (bits === 64 || !Number.isFinite(value)) return String(value);
  const exact = exactDecimal(f32);
  for (let p = 1; p <= 9; p++) {
    const parsed =
      exact.digits.length === p + 1 && exact.digits.endsWith("5")
        ? roundTieToEven(exact, p)
        : parseFloat(f32.toPrecision(p));
    if (Math.fround(parsed) === f32) return String(parsed);
  }
  return String(f32);
};

interface ExactDecimal {
  negative: boolean;
  digits: string;
  exponent: number;
}

// The exact value of a finite float32 as digits × 10^exponent, without trailing zeros.
const exactDecimal = (f32: number): ExactDecimal => {
  const view = new DataView(new ArrayBuffer(4));
  view.setFloat32(0, f32);
  const bits = view.getUint32(0);
  const biased = (bits >>> 23) & 0xff;
  const fraction = bits & 0x7fffff;
  const mantissa = BigInt(biased === 0 ? fraction : fraction | 0x800000);
  const power = (biased === 0 ? 1 : biased) - 150;
  let digits =
    power >= 0
      ? (mantissa << BigInt(power)).toString()
      : (mantissa * 5n ** BigInt(-power)).toString();
  let exponent = Math.min(power, 0);
  const trimmed = digits.replace(/0+$/, "");
  exponent += digits.length - trimmed.length;
  digits = trimmed;
  return { negative: bits >>> 31 === 1, digits, exponent };
};

// toPrecision rounds a tie up. The other runtimes round it to the even digit.
const roundTieToEven = ({ negative, digits, exponent }: ExactDecimal, p: number) => {
  const head = BigInt(digits.slice(0, p));
  const even = head % 2n === 0n ? head : head + 1n;
  return parseFloat(`${negative ? "-" : ""}${even}e${exponent + 1}`);
};
