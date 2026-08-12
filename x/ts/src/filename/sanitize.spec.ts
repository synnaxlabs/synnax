// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { sanitize } from "@/filename/sanitize";

describe("sanitize", () => {
  it("returns ordinary names unchanged", () => {
    expect(sanitize("My Project", "")).toEqual("My Project");
    expect(sanitize("metrics_2025", "")).toEqual("metrics_2025");
    expect(sanitize("name-with-dashes", "")).toEqual("name-with-dashes");
    expect(sanitize("file.json", "")).toEqual("file.json");
  });

  it("replaces forward and back slashes", () => {
    expect(sanitize("a/b", "")).toEqual("a_b");
    expect(sanitize("a\\b", "")).toEqual("a_b");
    expect(sanitize("path/to\\file", "")).toEqual("path_to_file");
  });

  it("replaces Windows-reserved characters", () => {
    expect(sanitize('a<b>c:d"e|f?g*h', "")).toEqual("a_b_c_d_e_f_g_h");
  });

  it("replaces the control characters Windows forbids", () => {
    expect(sanitize("a\x00b\x1fc", "")).toEqual("a_b_c");
  });

  it("leaves the delete character, which Windows allows, alone", () => {
    expect(sanitize("a\x7fb", "")).toEqual("a\x7fb");
  });

  it("drops trailing dots and spaces", () => {
    expect(sanitize("report.", "")).toEqual("report");
    expect(sanitize("report ", "")).toEqual("report");
    expect(sanitize("report. . ", "")).toEqual("report");
  });

  it("prefixes an underscore to a Windows device name", () => {
    expect(sanitize("CON", "")).toEqual("_CON");
    expect(sanitize("nul", "")).toEqual("_nul");
    expect(sanitize("COM1", "")).toEqual("_COM1");
    expect(sanitize("lpt9", "")).toEqual("_lpt9");
    expect(sanitize("aux.tar", "")).toEqual("_aux.tar");
  });

  it("leaves a name that only starts with a device name alone", () => {
    expect(sanitize("console", "")).toEqual("console");
    expect(sanitize("com10", "")).toEqual("com10");
  });

  it("names a file that sanitizes to nothing with an underscore", () => {
    expect(sanitize("", "")).toEqual("_");
    expect(sanitize(". . ", "")).toEqual("_");
  });

  it("does not collapse repeated unsafe characters", () => {
    expect(sanitize("a///b", "")).toEqual("a___b");
  });

  describe("extension", () => {
    it("carries the extension", () => {
      expect(sanitize("in/let", ".json")).toEqual("in_let.json");
    });

    it("names the file with an underscore when only the extension survives", () => {
      expect(sanitize("...", ".json")).toEqual("_.json");
    });

    it("throws when the extension fills a file name by itself", () => {
      expect(() => sanitize("report", "a".repeat(255))).toThrow(
        "leaves no room for a file name",
      );
    });
  });

  describe("length", () => {
    it("shortens a name too long for a file name", () => {
      expect(sanitize("a".repeat(400), "")).toEqual("a".repeat(255));
    });

    it("holds the extension's bytes back", () => {
      expect(sanitize("a".repeat(400), ".json")).toEqual(
        `${"a".repeat(255 - ".json".length)}.json`,
      );
    });

    it("cuts on a code point boundary", () => {
      // Each code point takes two bytes, so an odd limit cannot be filled exactly.
      expect(sanitize("é".repeat(200), "")).toEqual("é".repeat(127));
    });

    it("cuts without splitting a surrogate pair", () => {
      // Each emoji takes four bytes, so the limit lands mid-emoji.
      const sanitized = sanitize("🛰".repeat(100), "");
      expect(sanitized).toEqual("🛰".repeat(63));
      expect(sanitized).not.toContain("�");
    });

    it("drops a trailing space the cut exposes", () => {
      expect(sanitize(`${"a".repeat(254)} b`, "")).toEqual("a".repeat(254));
    });

    it("holds a byte back for a device name's prefix", () => {
      const sanitized = sanitize(`nul.${"a".repeat(400)}`, "");
      expect(sanitized).toHaveLength(255);
      expect(sanitized.startsWith("_nul.")).toBe(true);
    });
  });
});
