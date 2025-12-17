// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { csv } from "@/csv";

describe("csv", () => {
  describe("formatValue", () => {
    describe("numbers", () => {
      it("should format numbers correctly", () => {
        expect(csv.formatValue(123)).toBe("123");
        expect(csv.formatValue(123.456)).toBe("123.456");
        expect(csv.formatValue(12e10)).toBe("120000000000");
        expect(csv.formatValue(Infinity)).toBe("Infinity");
        expect(csv.formatValue(-Infinity)).toBe("-Infinity");
        expect(csv.formatValue(NaN)).toBe("NaN");
        expect(csv.formatValue(+0)).toBe("0");
        expect(csv.formatValue(-0)).toBe("0");
        expect(csv.formatValue(12e100)).toBe("1.2e+101");
      });
    });
    describe("booleans", () => {
      it("should format booleans correctly", () => {
        expect(csv.formatValue(true)).toBe("1");
        expect(csv.formatValue(false)).toBe("0");
      });
    });
    describe("strings", () => {
      it("should format strings correctly", () => {
        expect(csv.formatValue("hello")).toBe("hello");
        expect(csv.formatValue("hello,world")).toBe('"hello,world"');
        expect(csv.formatValue("hello\nworld")).toBe('"hello\nworld"');
        expect(csv.formatValue("hello\r\nworld")).toBe('"hello\r\nworld"');
        expect(csv.formatValue('"hello"')).toBe('"""hello"""');
      });
    });
    describe("symbols", () => {
      it("should format symbols correctly", () => {
        expect(csv.formatValue(Symbol("hello"))).toBe("Symbol(hello)");
        expect(csv.formatValue(Symbol('"hello"'))).toBe('"Symbol(""hello"")"');
      });
    });
    describe("functions", () => {
      it("should format functions correctly", () => {
        expect(csv.formatValue(() => "hello")).toBe('"() => ""hello"""');
      });
    });
    describe("null", () => {
      it("should format null correctly", () => {
        expect(csv.formatValue(null)).toBe("");
      });
    });
    describe("undefined", () => {
      it("should format undefined correctly", () => {
        expect(csv.formatValue(undefined)).toBe("");
      });
    });
    describe("objects", () => {
      it("should format objects correctly", () => {
        expect(csv.formatValue({ a: 1, b: 2 })).toBe('"{""a"":1,""b"":2}"');
      });
    });
    describe("arrays", () => {
      it("should format arrays correctly", () => {
        expect(csv.formatValue([1, 2, 3])).toBe('"[1,2,3]"');
      });
    });
  });
});
