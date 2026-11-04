// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { search } from "@/search";

describe("search", () => {
  describe("Index", () => {
    let idx: search.Index<number>;
    beforeEach(() => {
      idx = new search.Index<number>();
    });

    describe("matching", () => {
      it.each([
        ["an exact word", "test", "test"],
        ["a word in a multi-word entry", "October 28 Gooster", "Gooster"],
        ["a term with underscores", "gse_ai_15", "ai_15"],
        ["one replaced rune in capitals", "BBTPC", "BTTPC"],
        ["a prefix", "Channel", "ch"],
        ["a prefix of words with underscores", "DAQ_PT_1", "DAQ_PT"],
        ["one inserted rune", "sensor", "sensors"],
        ["one deleted rune", "sensor", "senor"],
        ["one replaced multi-byte rune", "résumé", "resumé"],
        ["a word split by '-'", "gse-ai", "ai"],
        ["a word split by '.'", "gse.ai", "ai"],
        ["a word split by a tab", "gse\tai", "ai"],
        ["a word split by '/'", "plant/line2/temp", "line2"],
        ["a capital 'İ' as an 'i'", "İl", "il"],
        ["a final capital 'Σ' as a 'σ'", "ΟΣ", "οσ"],
        ["a word split by a next line", "plant\u0085temp", "temp"],
        [
          "a word at the end of a long name",
          "plant/area_one/line_two/station_three/sensor_temperature",
          "temperature",
        ],
      ])("should match %s", (_, text, term) => {
        idx.set(1, text);
        expect(idx.search(term)).toEqual([1]);
      });
    });

    describe("not matching", () => {
      it.each([
        ["a different word", "test", "nope"],
        ["a short term two edits away", "Channel", "nn"],
        ["an empty term", "test", ""],
        ["a term of only separators", "test", " _-./"],
        ["a word joined by a zero width no-break space", "plant\uFEFFtemp", "temp"],
      ])("should not match %s", (_, text, term) => {
        idx.set(1, text);
        expect(idx.search(term)).toEqual([]);
      });
    });

    it("should not match a word of one or two runes one edit away", () => {
      idx.set(1, "sensor_1");
      idx.set(2, "sensor_2");
      idx.set(3, "October 29");
      idx.set(4, "October 28");
      expect(idx.search("sensor_2")).toEqual([2]);
      expect(idx.search("October 28")).toEqual([4]);
    });

    it("should rank an exact word, then a prefix, then a near word", () => {
      idx.set(1, "tost");
      idx.set(2, "testing");
      idx.set(3, "test");
      expect(idx.search("test")).toEqual([3, 2, 1]);
    });

    it("should rank fewer words first on a tie", () => {
      idx.set(1, "pressure feed");
      idx.set(2, "pressure");
      expect(idx.search("pressure")).toEqual([2, 1]);
    });

    it("should keep the closest matches up to the limit", () => {
      idx.set(1, "pressure");
      idx.set(2, "pressure feed");
      idx.set(3, "pressures");
      expect(idx.search("pressure", 2)).toEqual([1, 2]);
    });

    it("should match words across several texts", () => {
      idx.set(1, "My Device", "LabJack", "T7");
      expect(idx.search("device t7")).toEqual([1]);
    });

    it("should return only the entries that match every word", () => {
      idx.set(1, "Pressure Sensor", "NI");
      idx.set(2, "Pressure");
      expect(idx.search("Pressure Sensor")).toEqual([1]);
    });

    it("should return the entries that match any word when none match all", () => {
      idx.set(1, "My Blob");
      idx.set(2, "Other");
      expect(idx.search("blob nope")).toEqual([1]);
    });

    it("should read only the first 64 words of a term", () => {
      idx.set(1, "alpha");
      expect(idx.search(`${"zzz ".repeat(63)}alpha`)).toEqual([1]);
      expect(idx.search(`${"zzz ".repeat(64)}alpha`)).toEqual([]);
    });

    it("should replace the text of a key set again", () => {
      idx.set(1, "alpha");
      idx.set(1, "beta");
      expect(idx.search("alpha")).toEqual([]);
      expect(idx.search("beta")).toEqual([1]);
    });

    it("should not match a deleted key", () => {
      idx.set(1, "alpha");
      idx.delete(1);
      expect(idx.search("alpha")).toEqual([]);
    });
  });

  describe("matches", () => {
    it("should match any text when the term has no words", () => {
      expect(search.matches("Anything", " ")).toBe(true);
    });

    it("should match when every word of the term matches", () => {
      expect(search.matches("Create a new channel", "new chan")).toBe(true);
    });

    it("should not match when one word of the term does not match", () => {
      expect(search.matches("sensor_1", "sensor 9")).toBe(false);
    });
  });
});
