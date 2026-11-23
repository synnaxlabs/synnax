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

  it("should split identifiers only between words", () =>
    expect(strings.createShortIdentifiers("IOLITEr 32xDO BUS1")).toEqual([
      "i3b",
      "i_3_b",
      "i32xdob",
      "i_32xdo_b",
      "i3bus1",
      "i_3_bus1",
      "iol32xbus",
      "iol_32x_bus",
    ]));

  it("should leave out identifiers with characters other than letters, digits, and underscores", () =>
    expect(
      strings.createShortIdentifiers("IOLITE GATE LATCH v2.00 BUS1"),
    ).not.toContainEqual(expect.stringMatching(/[^\w]/)));
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

describe("escapeHTML", () => {
  it("should escape ampersands", () =>
    expect(strings.escapeHTML("a&b")).toBe("a&amp;b"));

  it("should escape angle brackets", () =>
    expect(strings.escapeHTML("<div>")).toBe("&lt;div&gt;"));

  it("should escape quotes", () =>
    expect(strings.escapeHTML(`"it's"`)).toBe("&quot;it&#39;s&quot;"));

  it("should return the original string when no special characters", () =>
    expect(strings.escapeHTML("hello")).toBe("hello"));

  it("should escape all special characters together", () =>
    expect(strings.escapeHTML(`<a href="x">&`)).toBe(
      "&lt;a href=&quot;x&quot;&gt;&amp;",
    ));
});

describe("toKebab", () => {
  const SPECS: [string, string][] = [
    ["fooBar", "foo-bar"],
    ["fooBarBaz", "foo-bar-baz"],
    ["foo bar", "foo-bar"],
    ["foo bar baz", "foo-bar-baz"],
    ["foo.bar", "foo.bar"],
    ["foo.bar.baz", "foo.bar.baz"],
    ["Foo Bar", "foo-bar"],
    ["foo_bar", "foo-bar"],
    ["foo_bar_baz", "foo-bar-baz"],
  ];
  SPECS.forEach(([input, expected]) => {
    it(`should convert ${input} to ${expected}`, () => {
      expect(strings.toKebab(input)).toBe(expected);
    });
  });
});
describe("toProperNoun", () => {
  const SPECS: [string, string][] = [
    ["fooBar", "Foo Bar"],
    ["foo_bar", "Foo Bar"],
    ["foo-bar", "Foo Bar"],
    ["FooBar", "Foo Bar"],
    ["foo_bar_baz", "Foo Bar Baz"],
    ["fooBarBaz", "Foo Bar Baz"],
    ["foo-bar-baz", "Foo Bar Baz"],
    ["XMLParser", "XML Parser"],
    ["parseXMLDocument", "Parse XML Document"],
    ["IODevice", "IO Device"],
    ["temperature_sensor", "Temperature Sensor"],
    ["pressure-gauge", "Pressure Gauge"],
    ["flowMeter", "Flow Meter"],
    ["my_custom_symbol", "My Custom Symbol"],
    ["valve-actuator-v2", "Valve Actuator V2"],
    ["PIDController", "PID Controller"],
    ["", ""],
    ["a", "A"],
    ["ABC", "ABC"],
    ["test123value", "Test123value"],
    ["test_123_value", "Test 123 Value"],
  ];
  SPECS.forEach(([input, expected]) => {
    it(`should convert ${input} to ${expected}`, () => {
      expect(strings.toProperNoun(input)).toBe(expected);
    });
  });
});

describe("toSentence", () => {
  const SPECS: [string, string][] = [
    ["sampleRate", "Sample rate"],
    ["state_update_rate", "State update rate"],
    ["data-type", "Data type"],
    ["FooBar", "Foo bar"],
    ["name", "Name"],
    ["XMLParser", "XML parser"],
    ["parseXMLDocument", "Parse XML document"],
    ["IODevice", "IO device"],
    ["apiKey", "Api key"],
    ["cjcSource", "Cjc source"],
    ["PIDController", "PID controller"],
    ["valve-actuator-v2", "Valve actuator v2"],
    ["", ""],
    ["_", ""],
    ["-", ""],
    [" ", ""],
    ["a", "A"],
    ["ABC", "ABC"],
    ["test_123_value", "Test 123 value"],
  ];
  SPECS.forEach(([input, expected]) => {
    it(`should convert ${input} to ${expected}`, () => {
      expect(strings.toSentence(input)).toBe(expected);
    });
  });
});

describe("capitalize", () => {
  it("should capitalize the first character", () => {
    expect(strings.capitalize("hello world")).toBe("Hello world");
  });

  it("should return an empty string unchanged", () => {
    expect(strings.capitalize("")).toBe("");
  });
});

describe("count", () => {
  it("should use the singular noun for a count of 1", () => {
    expect(strings.count(1, "cell")).toEqual("1 cell");
  });
  it("should add an s for any other count", () => {
    expect(strings.count(0, "cell")).toEqual("0 cells");
    expect(strings.count(3, "cell")).toEqual("3 cells");
  });
  it("should use the given plural", () => {
    expect(strings.count(2, "entry", "entries")).toEqual("2 entries");
    expect(strings.count(1, "entry", "entries")).toEqual("1 entry");
  });
});

describe("formatFloat", () => {
  it.each([
    [-1.5, "-1.5"],
    [1e10, "10000000000"],
    [1e20, "100000000000000000000"],
    [1e21, "1e+21"],
    [1e-6, "0.000001"],
    [1.5e-7, "1.5e-7"],
    [Number.MAX_VALUE, "1.7976931348623157e+308"],
    [5e-324, "5e-324"],
    [1000000000000000128, "1000000000000000100"],
    [0, "0"],
    [-0, "-0"],
    [NaN, "NaN"],
    [Infinity, "Infinity"],
    [-Infinity, "-Infinity"],
  ])("should format the float64 %s as %s", (value, expected) => {
    expect(strings.formatFloat(value)).toEqual(expected);
  });

  it.each([
    [0.1, "0.1"],
    [1e20, "100000000000000000000"],
    [1e30, "1e+30"],
    [1e-6, "0.000001"],
    [1.5e-7, "1.5e-7"],
    [0.000244140625, "0.00024414062"],
    [-0, "-0"],
    [NaN, "NaN"],
  ])("should format the float32 %s as %s", (value, expected) => {
    expect(strings.formatFloat(Math.fround(value), 32)).toEqual(expected);
  });

  it("should keep the sign of a value that rounds to float32 zero", () => {
    expect(strings.formatFloat(-1e-50, 32)).toEqual("-0");
  });

  it("should format text that parses back to the same bits", () => {
    const view = new DataView(new ArrayBuffer(8));
    let seed = 1;
    const next = (): number => {
      seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
      return seed;
    };
    for (let i = 0; i < 20_000; i++) {
      view.setUint32(0, next());
      view.setUint32(4, next());
      const f64 = view.getFloat64(0);
      if (!Number.isNaN(f64))
        expect(Object.is(Number(strings.formatFloat(f64)), f64)).toBe(true);
      view.setUint32(0, next());
      const f32 = view.getFloat32(0);
      if (!Number.isNaN(f32))
        expect(Object.is(Math.fround(Number(strings.formatFloat(f32, 32))), f32)).toBe(
          true,
        );
    }
  });
});
