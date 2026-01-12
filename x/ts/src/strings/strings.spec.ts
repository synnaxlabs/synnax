// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { strings } from "@/strings";

describe("naturalLanguageJoin", () => {
  it("should return an empty string for an empty array", () =>
    expect(strings.naturalLanguageJoin([])).toBe(""));

  it("should return the string for a single string", () =>
    expect(strings.naturalLanguageJoin("apple")).toBe("apple"));

  it("should return the zeroLength string for an empty array if provided", () =>
    expect(strings.naturalLanguageJoin([], "No items")).toBe("No items"));

  it("should return the single element for an array with one element", () =>
    expect(strings.naturalLanguageJoin(["apple"])).toBe("apple"));

  it('should join two elements with "and"', () =>
    expect(strings.naturalLanguageJoin(["apple", "banana"])).toBe("apple and banana"));

  it('should join multiple elements with commas and "and"', () =>
    expect(strings.naturalLanguageJoin(["apple", "banana", "cherry"])).toBe(
      "apple, banana, and cherry",
    ));

  it("should handle an array with more than three elements correctly", () =>
    expect(strings.naturalLanguageJoin(["apple", "banana", "cherry", "date"])).toBe(
      "apple, banana, cherry, and date",
    ));
});

describe("createShortIdentifiers", () => {
  it("should create identifiers for a single word", () =>
    expect(strings.createShortIdentifiers("Bob")).toEqual(
      expect.arrayContaining(["bob"]),
    ));

  it("should create identifiers for multiple words", () =>
    expect(strings.createShortIdentifiers("John Doe")).toEqual(
      expect.arrayContaining(["jd", "j_d", "johdoe", "joh_doe"]),
    ));

  it("should create identifiers for words containing numbers", () =>
    expect(strings.createShortIdentifiers("Alice 123")).toEqual(
      expect.arrayContaining(["a1", "a_1", "ali123", "ali_123"]),
    ));

  it("should create identifiers for words longer than three characters", () =>
    expect(strings.createShortIdentifiers("Jonathan")).toEqual(
      expect.arrayContaining(["jon"]),
    ));

  it("should create identifiers for words shorter than three characters", () =>
    expect(strings.createShortIdentifiers("Al")).toEqual(
      expect.arrayContaining(["al"]),
    ));

  it("should create identifiers for mixed cases", () =>
    expect(strings.createShortIdentifiers("Alice Bob")).toEqual(
      expect.arrayContaining(["ab", "a_b", "alibob", "ali_bob"]),
    ));
});

describe("trimPrefix", () => {
  it("should remove a prefix when it exists", () =>
    expect(strings.trimPrefix("hello world", "hello ")).toBe("world"));

  it("should return the original string when prefix does not exist", () =>
    expect(strings.trimPrefix("hello world", "goodbye ")).toBe("hello world"));

  it("should return the original string when prefix is empty", () =>
    expect(strings.trimPrefix("hello world", "")).toBe("hello world"));

  it("should return empty string when prefix equals the entire string", () =>
    expect(strings.trimPrefix("hello world", "hello world")).toBe(""));

  it("should handle case-sensitive matching", () =>
    expect(strings.trimPrefix("Hello World", "hello ")).toBe("Hello World"));

  it("should handle partial prefix matches", () =>
    expect(strings.trimPrefix("hello world", "hello")).toBe(" world"));

  it("should handle prefix longer than string", () =>
    expect(strings.trimPrefix("hello", "hello world")).toBe("hello"));

  it("should handle special characters in prefix", () =>
    expect(strings.trimPrefix("file:///path/to/file", "file://")).toBe(
      "/path/to/file",
    ));

  it("should handle unicode characters", () =>
    expect(strings.trimPrefix("café au lait", "café ")).toBe("au lait"));

  it("should handle numbers in prefix", () =>
    expect(strings.trimPrefix("123abc", "123")).toBe("abc"));
});
