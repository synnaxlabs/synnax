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

describe("Codec", () => {
  const sampleSchema = z.object({
    channelKey: z.string(),
    timeStamp: z.number(),
    value: z.unknown(),
  });

  binary.ENCODERS.forEach((e) => {
    describe(`encoder ${e.contentType}`, () => {
      it("should correctly encode and decode items", () => {
        const sample = {
          channelKey: "test",
          timeStamp: 123,
          value: [1, 2, 3],
        };
        const encoded = e.encode(sample);
        expect(e.decode(encoded, sampleSchema)).toEqual(sample);
      });
    });
  });

  describe("JSON", () => {
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

    it("should handle CSV with missing values", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [6],
      });
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

    it("should correctly encode array of objects to CSV", () => {
      const sampleData = [
        { name: "John", age: 30, city: "New York" },
        { name: "Alice", age: 25, city: "Boston" },
        { name: "Bob", age: 40, city: "Chicago" },
      ];

      const encoded = binary.CSV_CODEC.encodeString(sampleData);
      expect(encoded).toBe(
        'name,age,city\n"John",30,"New York"\n"Alice",25,"Boston"\n"Bob",40,"Chicago"',
      );
    });

    it("should handle objects with missing values", () => {
      const sampleData = [
        { name: "John", age: 30, city: "New York" },
        { name: "Alice", age: 25 },
        { name: "Bob", city: "Chicago" },
      ];

      const encoded = binary.CSV_CODEC.encodeString(sampleData);
      expect(encoded).toBe(
        'name,age,city\n"John",30,"New York"\n"Alice",25,""\n"Bob","","Chicago"',
      );
    });

    it("should handle objects with null and undefined values", () => {
      const sampleData = [
        { name: "John", age: null, city: undefined },
        { name: "Alice", age: 25, city: null },
      ];

      const encoded = binary.CSV_CODEC.encodeString(sampleData);
      expect(encoded).toBe('name,age,city\n"John","",""\n"Alice",25,""');
    });

    it("should handle different data types in objects", () => {
      const sampleData = [
        { name: "John", active: true, score: 98.5 },
        { name: "Alice", active: false, score: 92.3 },
      ];

      const encoded = binary.CSV_CODEC.encodeString(sampleData);
      expect(encoded).toBe('name,active,score\n"John",true,98.5\n"Alice",false,92.3');
    });

    it("should throw error when encoding empty array", () => {
      const sampleData: unknown[] = [];

      expect(() => {
        binary.CSV_CODEC.encodeString(sampleData);
      }).toThrow("Payload must be an array of objects");
    });

    it("should throw error when encoding non-array data", () => {
      const sampleData = { name: "John", age: 30 };

      expect(() => {
        binary.CSV_CODEC.encodeString(sampleData);
      }).toThrow("Payload must be an array of objects");
    });

    it("should round-trip encode and decode CSV data", () => {
      const sampleData = [
        { name: "John", age: 30, city: "New York" },
        { name: "Alice", age: 25, city: "Boston" },
      ];

      const encoded = binary.CSV_CODEC.encode(sampleData);
      const decoded = binary.CSV_CODEC.decode(encoded);

      expect(decoded).toEqual({
        name: ["John", "Alice"],
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
