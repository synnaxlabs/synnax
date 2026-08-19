// Copyright 2026 Synnax Labs, Inc.
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
  describe("parseBlock", () => {
    it("should split a tab delimited block into rows of fields", () => {
      expect(csv.parseBlock("1\t10\n2\t20")).toEqual([
        ["1", "10"],
        ["2", "20"],
      ]);
    });

    it("should split a comma delimited block", () => {
      expect(csv.parseBlock("1,10\n2,20")).toEqual([
        ["1", "10"],
        ["2", "20"],
      ]);
    });

    it("should keep commas inside a tab delimited field", () => {
      expect(csv.parseBlock("1,000\t2,000")).toEqual([["1,000", "2,000"]]);
    });

    it("should keep a delimiter inside a quoted field", () => {
      expect(csv.parseBlock('"a,b",c')).toEqual([["a,b", "c"]]);
    });

    it("should unescape doubled quotes", () => {
      expect(csv.parseBlock('"say ""hi""",b')).toEqual([['say "hi"', "b"]]);
    });

    it("should trim the fields", () => {
      expect(csv.parseBlock(" 1 , 10 ")).toEqual([["1", "10"]]);
    });

    it("should drop empty lines", () => {
      expect(csv.parseBlock("1,10\n\n2,20\n")).toEqual([
        ["1", "10"],
        ["2", "20"],
      ]);
    });

    it("should handle carriage returns", () => {
      expect(csv.parseBlock("1,10\r\n2,20")).toEqual([
        ["1", "10"],
        ["2", "20"],
      ]);
    });

    it("should leave ragged rows ragged", () => {
      expect(csv.parseBlock("1,10,100\n2")).toEqual([["1", "10", "100"], ["2"]]);
    });

    it("should keep a newline inside a quoted field", () => {
      expect(csv.parseBlock('"a\nb",c')).toEqual([["a\nb", "c"]]);
    });

    it("should split on commas when the only tab is inside a quoted field", () => {
      expect(csv.parseBlock('"a\tb",c')).toEqual([["a\tb", "c"]]);
    });

    it("should return no rows for empty text", () => {
      expect(csv.parseBlock("")).toEqual([]);
    });
  });

  describe("formatBlock", () => {
    it("should join rows into a tab delimited block", () => {
      expect(
        csv.formatBlock([
          [1, 10],
          [2, 20],
        ]),
      ).toBe("1\t10\n2\t20");
    });

    it("should quote a field holding a delimiter", () => {
      expect(csv.formatBlock([["a\tb", "c,d"]])).toBe('"a\tb"\t"c,d"');
    });

    it("should round-trip through parseBlock", () => {
      const rows = [
        ["a,b", 'say "hi"'],
        ["1", "2"],
      ];
      expect(csv.parseBlock(csv.formatBlock(rows))).toEqual(rows);
    });

    it("should round-trip a field holding a newline", () => {
      const rows = [["a\nb", "c"]];
      expect(csv.parseBlock(csv.formatBlock(rows))).toEqual(rows);
    });
  });
});
