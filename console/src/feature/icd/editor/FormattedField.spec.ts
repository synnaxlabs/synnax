// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  formatEscaped,
  formatHexBytes,
  parseEscaped,
  parseHexBytes,
} from "@/feature/icd/editor/FormattedField";

describe("formatHexBytes", () => {
  it("should write each byte as two uppercase digits split by spaces", () => {
    expect(formatHexBytes(new Uint8Array([0xaa, 0x05, 0xff]))).toBe("AA 05 FF");
  });
});

describe("parseHexBytes", () => {
  it.each([
    ["AA 55", [0xaa, 0x55]],
    ["aa55", [0xaa, 0x55]],
    ["0xAA 0x55", [0xaa, 0x55]],
    [" 7e\t01 ", [0x7e, 0x01]],
  ])("should parse %j", (text, want) => {
    expect(parseHexBytes(text)).toEqual(new Uint8Array(want));
  });

  it.each(["", "  ", "A", "AA 5", "GG", "0x"])("should refuse %j", (text) => {
    expect(parseHexBytes(text)).toBeNull();
  });

  it("should give back the bytes that formatHexBytes writes", () => {
    const bytes = new Uint8Array([0x00, 0x7e, 0xaa, 0xff]);
    expect(parseHexBytes(formatHexBytes(bytes))).toEqual(bytes);
  });
});

describe("formatEscaped", () => {
  it.each([
    ["\n", "\\n"],
    ["\r\n", "\\r\\n"],
    ["\t", "\\t"],
    ["\\", "\\\\"],
    ["\x00", "\\x00"],
    ["\x7f", "\\x7F"],
    ["END;", "END;"],
  ])("should write %j as %j", (text, want) => {
    expect(formatEscaped(text)).toBe(want);
  });
});

describe("parseEscaped", () => {
  it.each([
    ["\\n", "\n"],
    ["\\r\\n", "\r\n"],
    ["\\t", "\t"],
    ["\\\\", "\\"],
    ["\\x03", "\x03"],
    ["\\x7f", "\x7f"],
    ["END;", "END;"],
  ])("should parse %j", (text, want) => {
    expect(parseEscaped(text)).toBe(want);
  });

  it.each(["", "\\", "\\q", "\\x", "\\x8", "\\x80", "\\xZZ"])(
    "should refuse %j",
    (text) => {
      expect(parseEscaped(text)).toBeNull();
    },
  );

  it("should give back the text that formatEscaped writes", () => {
    const text = "\x02 \\ \r\n\t\x7f;";
    expect(parseEscaped(formatEscaped(text))).toBe(text);
  });
});
