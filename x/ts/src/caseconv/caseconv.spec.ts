// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { caseconv } from "@/caseconv";

describe("caseconv", () => {
  describe("snakeToCamel", () => {
    describe("strings", () => {
      const SPECS: [string, string][] = [
        ["foo_bar", "fooBar"],
        ["foo_bar_baz", "fooBarBaz"],
        ["foo", "foo"],
        ["fooBar", "fooBar"],
        ["fooBarBaz", "fooBarBaz"],
        ["NS=1;ID=5", "NS=1;ID=5"],
        ["foo-bar", "foo-bar"],
        ["FooBar", "fooBar"],
        ["foo.bar", "foo.bar"],
        ["foo_bar_baz.qux", "fooBarBaz.qux"],
      ];
      SPECS.forEach(([input, expected]) => {
        it(`should convert ${input} to ${expected}`, () => {
          expect(caseconv.snakeToCamel(input)).toBe(expected);
        });
      });
    });
    describe("objects", () => {
      const SPECS: [unknown, unknown][] = [
        [{ foo_bar: 1 }, { fooBar: 1 }],
        [{ foo_bar_baz: 1 }, { fooBarBaz: 1 }],
        [{ foo: 1 }, { foo: 1 }],
        [{ fooBar: 1 }, { fooBar: 1 }],
        [{ fooBarBaz: 1 }, { fooBarBaz: 1 }],
        [
          { NS: 1, ID: 5 },
          { NS: 1, ID: 5 },
        ],
        [{ "foo-bar": 1 }, { "foo-bar": 1 }],
        [{ foo_bar: { baz_qux: 1 } }, { fooBar: { bazQux: 1 } }],
        [[{ foo_bar: 1 }], [{ fooBar: 1 }]],
        [
          [{ foo_bar: 1 }, { baz_qux: 2 }],
          [{ fooBar: 1 }, { bazQux: 2 }],
        ],
        [
          { channel_key: "test", time_stamp: 123, value: [1, 2, 3] },
          { channelKey: "test", timeStamp: 123, value: [1, 2, 3] },
        ],
      ];
      SPECS.forEach(([input, expected], i) => {
        it(`should convert object ${i}`, () => {
          expect(caseconv.snakeToCamel(input)).toEqual(expected);
        });
      });
    });
  });
  describe("camelToSnake", () => {
    describe("strings", () => {
      const SPECS: [string, string][] = [
        ["fooBar", "foo_bar"],
        ["fooBarBaz", "foo_bar_baz"],
        ["foo", "foo"],
        ["foo_bar", "foo_bar"],
        ["foo_bar_baz", "foo_bar_baz"],
        ["NS=1;ID=5", "NS=1;ID=5"],
        ["foo-bar", "foo-bar"],
        ["foo.bar", "foo.bar"],
        ["fooBarBaz.qux", "foo_bar_baz.qux"],
      ];
      SPECS.forEach(([input, expected]) => {
        it(`should convert ${input} to ${expected}`, () => {
          expect(caseconv.camelToSnake(input)).toBe(expected);
        });
      });
    });
    describe("objects", () => {
      const SPECS: [unknown, unknown][] = [
        [{ fooBar: 1 }, { foo_bar: 1 }],
        [{ fooBarBaz: 1 }, { foo_bar_baz: 1 }],
        [{ foo: 1 }, { foo: 1 }],
        [{ foo_bar: 1 }, { foo_bar: 1 }],
        [{ foo_bar_baz: 1 }, { foo_bar_baz: 1 }],
        [
          { NS: 1, ID: 5 },
          { NS: 1, ID: 5 },
        ],
        [{ "foo-bar": 1 }, { "foo-bar": 1 }],
        [{ fooBar: { bazQux: 1 } }, { foo_bar: { baz_qux: 1 } }],
        [[{ fooBar: 1 }], [{ foo_bar: 1 }]],
        [
          [{ fooBar: 1 }, { bazQux: 2 }],
          [{ foo_bar: 1 }, { baz_qux: 2 }],
        ],
        [
          { channelKey: "test", timeStamp: 123, value: [1, 2, 3] },
          { channel_key: "test", time_stamp: 123, value: [1, 2, 3] },
        ],
      ];
      SPECS.forEach(([input, expected], i) => {
        it(`should convert object ${i}`, () => {
          expect(caseconv.camelToSnake(input)).toEqual(expected);
        });
      });
    });
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
        expect(caseconv.toKebab(input)).toBe(expected);
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
        expect(caseconv.toProperNoun(input)).toBe(expected);
      });
    });
  });
});
