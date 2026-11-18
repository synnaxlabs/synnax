// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";
import { color } from "@/color";
import { zod } from "@/zod";

describe("Codec", () => {
  const sampleSchema = z.object({
    channelKey: z.string(),
    timeStamp: z.number(),
    value: z.unknown(),
  });

  describe("JSON", () => {
    it("should correctly encode and decode items", () => {
      const sample = {
        channelKey: "test",
        timeStamp: 123,
        value: [1, 2, 3],
      };
      const encoded = binary.JSON_CODEC.encode(sample);
      expect(binary.JSON_CODEC.decode(encoded, sampleSchema)).toEqual(sample);
    });

    it("should decode a wire color to a tuple", () => {
      const schema = z.object({ fillColor: color.colorZ });
      const decoded = new binary.JSONCodec().decodeString(
        JSON.stringify({ fill_color: "#ff000080" }),
        schema,
      );
      expect(decoded.fillColor).toEqual([255, 0, 0, 128]);
    });

    it("should reject a bad wire color with a labeled parse error", () => {
      const schema = z.object({ fillColor: color.colorZ });
      expect(() =>
        new binary.JSONCodec().decodeString(
          JSON.stringify({ fill_color: [1, 2, 3, 0.5, 9] }),
          schema,
        ),
      ).toThrow(zod.ParseError);
    });

    it("should correctly convert keys to snake case", () => {
      const sample = {
        channelKey: "test",
        timeStamp: 123,
        value: new Array([1, 2, 3]),
      };
      const encoded = binary.JSON_CODEC.encodeString(sample);
      const parse = JSON.parse(encoded);
      expect(parse.channel_key).toEqual("test");
    });

    it("should correctly decode keys from snake case", () => {
      const sample = {
        channel_key: "test",
        time_stamp: 123,
        value: new Array([1, 2, 3]),
      };
      const encoded = JSON.stringify(sample);
      const decoded = binary.JSON_CODEC.decodeString(encoded, sampleSchema);
      expect(decoded.channelKey).toEqual("test");
    });

    it("should carry the caller's label into parse errors when decoding", () => {
      const schema = z.object({ name: z.string() });
      const encoded = JSON.stringify({ name: 42 });
      expect(() =>
        binary.JSON_CODEC.decodeString(encoded, schema, { label: "/range/create" }),
      ).toThrow("Failed to parse /range/create");
    });

    it("should validate with schema when encoding", () => {
      const schema = z.object({
        name: z.string(),
        count: z.number(),
      });
      const sample = { name: "test", count: 42 };
      const encoded = binary.JSON_CODEC.encodeString(sample, schema);
      const parsed = JSON.parse(encoded);
      expect(parsed.name).toEqual("test");
      expect(parsed.count).toEqual(42);
    });

    it("should throw on invalid data when schema is provided to encode", () => {
      const schema = z.object({
        name: z.string(),
        count: z.number(),
      });
      const sample = { name: "test", count: "not a number" };
      expect(() => binary.JSON_CODEC.encodeString(sample, schema)).toThrow();
    });

    it("should apply transforms when encoding with schema", () => {
      const schema = z.object({
        value: z.coerce.number(),
      });
      const sample = { value: "42" };
      const encoded = binary.JSON_CODEC.encodeString(sample, schema);
      const parsed = JSON.parse(encoded);
      expect(parsed.value).toEqual(42);
    });
  });

  describe("CSV", () => {
    it("should correctly decode CSV data with valid input", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,5
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [5, 6],
      });
    });

    it("should handle empty CSV data", () => {
      const sample = `
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({});
    });

    it("should handle CSV with only headers", () => {
      const sample = `
      channelKey,timeStamp,value
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: [],
        timeStamp: [],
        value: [],
      });
    });

    it("should decode an empty cell to null in its row", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,
      test2,,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, null],
        value: [null, 6],
      });
    });

    it("should decode a missing cell at the end of a short row to null", () => {
      const sample = "a,b,c\n1,2\n4,5,6";
      expect(binary.CSV_CODEC.decodeString(sample)).toEqual({
        a: [1, 4],
        b: [2, 5],
        c: [null, 6],
      });
    });

    it("should keep a delimiter inside a quoted field", () => {
      const sample = 'name,city\n"Doe, John","Paris, France"';
      expect(binary.CSV_CODEC.decodeString(sample)).toEqual({
        name: ["Doe, John"],
        city: ["Paris, France"],
      });
    });

    it("should decode a quoted number to a number", () => {
      const sample = 'raw,scaled\n"1.5","10"\n2,"20"';
      expect(binary.CSV_CODEC.decodeString(sample)).toEqual({
        raw: [1.5, 2],
        scaled: [10, 20],
      });
    });

    it("should split on commas when a field holds a tab", () => {
      expect(binary.CSV_CODEC.decodeString("raw,scaled\n1,\t10")).toEqual({
        raw: [1],
        scaled: [10],
      });
    });

    it("should keep a quoted field of spaces as a string", () => {
      expect(binary.CSV_CODEC.decodeString('a,b\n" ",1')).toEqual({
        a: [" "],
        b: [1],
      });
    });

    it("should parse the decoded columns with the schema", () => {
      const schema = z.object({ a: z.array(z.number()) });
      expect(binary.CSV_CODEC.decodeString("a\n1\n2", schema)).toEqual({ a: [1, 2] });
      expect(() => binary.CSV_CODEC.decodeString("a\nx", schema)).toThrow(
        zod.ParseError,
      );
    });

    it("should handle CSV with extra values", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,5,extra
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [5, 6],
      });
    });

    it("should handle CSV with different types of values", () => {
      const sample = `
      key,number,string
      test,123,"hello"
      test2,456,"world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with spaces around values", () => {
      const sample = `
      key, number , string
      test , 123 , "hello"
      test2 , 456 , "world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with empty rows", () => {
      const sample = `
      key,number,string
      test,123,"hello"
      ,
      test2,456,"world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with single column", () => {
      const sample = `
      key
      test
      test2
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
      });
    });

    const encodeToString = (payload: unknown): string =>
      new TextDecoder().decode(binary.CSV_CODEC.encode(payload));

    it("should correctly encode array of objects to CSV", () => {
      const sampleData = [
        { name: "John", age: 30, city: "New York" },
        { name: "Alice", age: 25, city: "Boston" },
        { name: "Bob", age: 40, city: "Chicago" },
      ];
      expect(encodeToString(sampleData)).toBe(
        "name,age,city\nJohn,30,New York\nAlice,25,Boston\nBob,40,Chicago",
      );
    });

    it("should handle objects with missing values", () => {
      const sampleData = [
        { name: "John", age: 30, city: "New York" },
        { name: "Alice", age: 25 },
        { name: "Bob", city: "Chicago" },
      ];
      expect(encodeToString(sampleData)).toBe(
        "name,age,city\nJohn,30,New York\nAlice,25,\nBob,,Chicago",
      );
    });

    it("should handle objects with null and undefined values", () => {
      const sampleData = [
        { name: "John", age: null, city: undefined },
        { name: "Alice", age: 25, city: null },
      ];
      expect(encodeToString(sampleData)).toBe("name,age,city\nJohn,,\nAlice,25,");
    });

    it("should handle different data types in objects", () => {
      const sampleData = [
        { name: "John", active: true, score: 98.5 },
        { name: "Alice", active: false, score: 92.3 },
      ];
      expect(encodeToString(sampleData)).toBe(
        "name,active,score\nJohn,1,98.5\nAlice,0,92.3",
      );
    });

    it("should quote fields by CSV rules", () => {
      const sampleData = [{ "a,b": 'say "hi"', c: "x,y" }];
      expect(encodeToString(sampleData)).toBe('"a,b",c\n"say ""hi""","x,y"');
    });

    it("should throw error when encoding empty array", () => {
      expect(() => binary.CSV_CODEC.encode([])).toThrow(
        "Payload must be an array of objects",
      );
    });

    it("should throw error when encoding non-array data", () => {
      expect(() => binary.CSV_CODEC.encode({ name: "John", age: 30 })).toThrow(
        "Payload must be an array of objects",
      );
    });

    it("should round-trip encode and decode CSV data", () => {
      const sampleData = [
        { name: "Doe, John", age: 30, city: "New York" },
        { name: 'Alice "Al"', age: 25, city: "Boston" },
      ];

      const encoded = binary.CSV_CODEC.encode(sampleData);
      const decoded = binary.CSV_CODEC.decode(encoded);

      expect(decoded).toEqual({
        name: ["Doe, John", 'Alice "Al"'],
        age: [30, 25],
        city: ["New York", "Boston"],
      });
    });
  });

  describe("Text", () => {
    it("should correctly encode and decode text", () => {
      const sampleText = "Hello, world!";
      const encoded = binary.TEXT_CODEC.encode(sampleText);
      const decoded = binary.TEXT_CODEC.decode(encoded);
      expect(decoded).toEqual(sampleText);
    });

    it("should handle empty strings", () => {
      const sampleText = "";
      const encoded = binary.TEXT_CODEC.encode(sampleText);
      const decoded = binary.TEXT_CODEC.decode(encoded);
      expect(decoded).toEqual(sampleText);
    });

    it("should validate text with schema", () => {
      const textSchema = z.string();
      const sampleText = "Validation test";
      const encoded = binary.TEXT_CODEC.encode(sampleText);
      const decoded = binary.TEXT_CODEC.decode(encoded, textSchema);
      expect(decoded).toEqual(sampleText);
    });

    it("should handle special characters", () => {
      const sampleText = "Special characters: äöü!@#$%^&*()_+";
      const encoded = binary.TEXT_CODEC.encode(sampleText);
      const decoded = binary.TEXT_CODEC.decode(encoded);
      expect(decoded).toEqual(sampleText);
    });

    it("should handle multi-line text", () => {
      const sampleText = "Line 1\nLine 2\nLine 3";
      const encoded = binary.TEXT_CODEC.encode(sampleText);
      const decoded = binary.TEXT_CODEC.decode(encoded);
      expect(decoded).toEqual(sampleText);
    });
  });
});
