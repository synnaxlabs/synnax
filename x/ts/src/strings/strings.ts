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
 * @param zeroLength - The string to return if the array is empty. Defaults to an empty
 * string.
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
  const [first, second, ...rest] = strings;
  if (first == null) return zeroLength;
  if (second == null) return first;
  const last = rest.pop();
  if (last == null) return `${first} and ${second}`;
  return `${[first, second, ...rest].join(", ")}, and ${last}`;
};

/**
 * Creates a list of short identifiers from a given name.
 *
 * @param name - The name to create identifiers from.
 * @returns An array of unique short identifiers. Each starts with a letter and holds
 * only letters, digits, and underscores.
 *
 * @example
 * ```typescript
 * createShortIdentifiers("John Doe"); // ["jd", "j_d", "johdoe", "joh_doe"]
 * createShortIdentifiers("Alice 123");
 * // ["a1", "a_1", "a123", "a_123", "ali123", "ali_123"]
 * createShortIdentifiers("Bob"); // ["bob"]
 * ```
 */
export const createShortIdentifiers = (name: string): string[] => {
  const words = name
    .split(" ")
    .filter((word) => word !== "")
    .map((word) => word.toLowerCase().replace(/-/g, "_"));
  const identifiers = new Set<string>();
  const add = (parts: string[]) => {
    identifiers.add(parts.join(""));
    identifiers.add(parts.join("_"));
  };
  add(words.map((word) => word.charAt(0)));
  words.forEach((word, index) => {
    if (/\d/.test(word)) add(words.map((w, i) => (i === index ? w : w.charAt(0))));
  });
  add(words.map((word) => word.substring(0, 3)));
  return Array.from(identifiers).filter(
    (id) => id.length >= 2 && id.length <= 12 && /^[a-zA-Z]\w*$/.test(id),
  );
};

/**
 * Removes a prefix from a string if it exists.
 *
 * @param str - The string to remove the prefix from.
 * @param prefix - The prefix to remove.
 * @returns The string with the prefix removed if it was present, otherwise the original
 * string.
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

/** Capitalizes the first character of the given string. */
export const capitalize = (str: string): string => {
  if (str.length === 0) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
};

/** Converts a string to kebab-case, splitting on spaces, camelCase, and capitals. */
export const toKebab = (str: string): string =>
  str
    .replace(/[\s_]+/g, "-")
    .replace(
      /([a-z0-9])([A-Z])/g,
      (_, p1: string, p2: string) => `${p1}-${p2.toLowerCase()}`,
    )
    .toLowerCase();

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

/** Converts a string to proper noun format, capitalizing every word. */
export const toProperNoun = (str: string): string => {
  if (str.length === 0) return str;
  return splitWords(str).replace(/\b\w/g, (char) => char.toUpperCase());
};

const ACRONYM_REGEX = /^[A-Z0-9]{2,}$/;

/** Converts a string to sentence case. Acronyms keep their casing. */
export const toSentence = (str: string): string => {
  const spaced = splitWords(str);
  if (spaced.length === 0) return spaced;
  const words = spaced.split(" ");
  return words
    .map((word, i) => {
      if (ACRONYM_REGEX.test(word)) return word;
      const lower = word.toLowerCase();
      return i === 0 ? lower.charAt(0).toUpperCase() + lower.slice(1) : lower;
    })
    .join(" ");
};

/**
 * Formats a count with a noun that agrees with it.
 *
 * @param n - The count.
 * @param noun - The singular noun.
 * @param plural - The plural noun. Defaults to the noun with an "s" added.
 * @returns the count followed by the noun, singular only when the count is 1.
 *
 * @example
 * ```typescript
 * count(1, "cell"); // "1 cell"
 * count(3, "cell"); // "3 cells"
 * count(0, "entry", "entries"); // "0 entries"
 * ```
 */
export const count = (n: number, noun: string, plural: string = `${noun}s`): string =>
  `${n} ${n === 1 ? noun : plural}`;
