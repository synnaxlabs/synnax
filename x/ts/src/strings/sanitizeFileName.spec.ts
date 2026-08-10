// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { sanitizeFileName } from "@/strings/sanitizeFileName";

describe("sanitizeFileName", () => {
  it("returns ordinary names unchanged", () => {
    expect(sanitizeFileName("My Project")).toEqual("My Project");
    expect(sanitizeFileName("metrics_2025")).toEqual("metrics_2025");
    expect(sanitizeFileName("name-with-dashes")).toEqual("name-with-dashes");
    expect(sanitizeFileName("file.json")).toEqual("file.json");
  });

  it("replaces forward and back slashes", () => {
    expect(sanitizeFileName("a/b")).toEqual("a_b");
    expect(sanitizeFileName("a\\b")).toEqual("a_b");
    expect(sanitizeFileName("path/to\\file")).toEqual("path_to_file");
  });

  it("replaces Windows-reserved characters", () => {
    expect(sanitizeFileName('a<b>c:d"e|f?g*h')).toEqual("a_b_c_d_e_f_g_h");
  });

  it("replaces the control characters Windows forbids", () => {
    expect(sanitizeFileName("a\x00b\x1fc")).toEqual("a_b_c");
  });

  it("leaves the delete character, which Windows allows, alone", () => {
    expect(sanitizeFileName("a\x7fb")).toEqual("a\x7fb");
  });

  it("drops trailing dots and spaces", () => {
    expect(sanitizeFileName("report.")).toEqual("report");
    expect(sanitizeFileName("report ")).toEqual("report");
    expect(sanitizeFileName("report. . ")).toEqual("report");
  });

  it("prefixes an underscore to a Windows device name", () => {
    expect(sanitizeFileName("CON")).toEqual("_CON");
    expect(sanitizeFileName("nul")).toEqual("_nul");
    expect(sanitizeFileName("COM1")).toEqual("_COM1");
    expect(sanitizeFileName("lpt9")).toEqual("_lpt9");
    expect(sanitizeFileName("aux.tar")).toEqual("_aux.tar");
  });

  it("leaves a name that only starts with a device name alone", () => {
    expect(sanitizeFileName("console")).toEqual("console");
    expect(sanitizeFileName("com10")).toEqual("com10");
  });

  it("names a file that sanitizes to nothing with an underscore", () => {
    expect(sanitizeFileName("")).toEqual("_");
    expect(sanitizeFileName(". . ")).toEqual("_");
  });

  it("does not collapse repeated unsafe characters", () => {
    expect(sanitizeFileName("a///b")).toEqual("a___b");
  });

  describe("extension", () => {
    it("carries the extension", () => {
      expect(sanitizeFileName("in/let", ".json")).toEqual("in_let.json");
    });

    it("names the file with an underscore when only the extension survives", () => {
      expect(sanitizeFileName("...", ".json")).toEqual("_.json");
    });

    it("throws when the extension fills a file name by itself", () => {
      expect(() => sanitizeFileName("report", "a".repeat(255))).toThrow(
        "leaves no room for a file name",
      );
    });
  });

  describe("length", () => {
    it("shortens a name too long for a file name", () => {
      expect(sanitizeFileName("a".repeat(400))).toEqual("a".repeat(255));
    });

    it("holds the extension's bytes back", () => {
      expect(sanitizeFileName("a".repeat(400), ".json")).toEqual(
        `${"a".repeat(255 - ".json".length)}.json`,
      );
    });

    it("cuts on a code point boundary", () => {
      // Each code point takes two bytes, so an odd limit cannot be filled exactly.
      expect(sanitizeFileName("é".repeat(200))).toEqual("é".repeat(127));
    });

    it("cuts without splitting a surrogate pair", () => {
      // Each emoji takes four bytes, so the limit lands mid-emoji.
      const sanitized = sanitizeFileName("🛰".repeat(100));
      expect(sanitized).toEqual("🛰".repeat(63));
      expect(sanitized).not.toContain("�");
    });

    it("drops a trailing space the cut exposes", () => {
      expect(sanitizeFileName(`${"a".repeat(254)} b`)).toEqual("a".repeat(254));
    });

    it("holds a byte back for a device name's prefix", () => {
      const sanitized = sanitizeFileName(`nul.${"a".repeat(400)}`);
      expect(sanitized).toHaveLength(255);
      expect(sanitized.startsWith("_nul.")).toBe(true);
    });
  });
});
