// Copyright 2025 Synnax Labs, Inc.
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
  strings: string[],
  zeroLength: string = "",
): string => {
  const length = strings.length;
  if (length === 0) return zeroLength;
  if (length === 1) return strings[0];
  if (length === 2) return `${strings[0]} and ${strings[1]}`;
  return `${strings.slice(0, -1).join(", ")}, and ${strings[length - 1]}`;
};

/**
 * Generates a list of short identifiers from a given name.
 *
 * @param name - The name to generate identifiers from.
 * @returns An array of unique short identifiers.
 *
 * @example
 * ```typescript
 * generateShortIdentifiers("John Doe"); // ["jd", "j_d", "johdoe", "joh_doe"]
 * generateShortIdentifiers("Alice 123"); // ["a1", "a_1", "a123", "a_12_3", "ali123", "ali_123"]
 * generateShortIdentifiers("Bob"); // ["bob"]
 * ```
 */
export const generateShortIdentifiers = (name: string): string[] => {
  const words = name.split(" ");
  const identifiers = new Set<string>();

  // Generate initials
  const initials = words.map((word) => word.charAt(0).toLowerCase()).join("");
  identifiers.add(initials.replace(/-/g, "_"));
  identifiers.add(initials.replace(/(.)(.)/g, "$1_$2").replace(/-/g, "_")); // Insert underscores

  // Generate combinations with numbers
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

  // Generate other potential combinations
  const wordAbbreviations = words.map((word) =>
    (word.length > 3 ? word.substring(0, 3) : word).toLowerCase(),
  );
  identifiers.add(wordAbbreviations.join("").replace(/-/g, "_"));
  identifiers.add(wordAbbreviations.join("_").replace(/-/g, "_"));

  // Limit length of identifiers and ensure they don't start with numbers
  const filteredIdentifiers = Array.from(identifiers).filter(
    (id) => id.length >= 2 && id.length <= 12 && !/^\d/.test(id),
  );

  return filteredIdentifiers;
};
