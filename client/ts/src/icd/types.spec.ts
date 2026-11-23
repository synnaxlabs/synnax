// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { icd } from "@/icd";

const ICD = { key: "7f1b3a52-8c1d-4f0e-9d5a-1a2b3c4d5e00", name: "Rig", messages: [] };

const parseFraming = (framing: unknown): icd.Framing | undefined =>
  icd.icdZ.parse({ ...ICD, framing }).framing;

describe("icdZ", () => {
  describe("framing", () => {
    it("should parse a null framing as absent", () => {
      expect(parseFraming(null)).toBeUndefined();
    });

    it("should parse a missing framing as absent", () => {
      expect(icd.icdZ.parse(ICD).framing).toBeUndefined();
    });

    const DEFAULTS: [string, unknown, icd.Framing][] = [
      ["delimited", { type: "delimited" }, { type: "delimited", delimiter: "\n" }],
      ["fixed", { type: "fixed" }, { type: "fixed", length: 1 }],
      [
        "sync",
        { type: "sync", sync: "qlU=" },
        {
          type: "sync",
          sync: new Uint8Array([0xaa, 0x55]),
          lengthOffset: 0,
          lengthSize: "one",
          byteOrder: "little_endian",
          lengthAdjustment: 0,
          maxLength: 65536,
        },
      ],
      ["cobs", { type: "cobs" }, { type: "cobs" }],
      ["slip", { type: "slip" }, { type: "slip" }],
    ];
    it.each(DEFAULTS)("should fill in the defaults of %s framing", (_, input, want) => {
      expect(parseFraming(input)).toEqual(want);
    });

    it("should fill in the byte order of a checksum", () => {
      const framing = parseFraming({
        type: "sync",
        sync: "fg==",
        checksum: { algorithm: "crc32" },
      });
      expect(framing).toMatchObject({
        checksum: { algorithm: "crc32", byteOrder: "little_endian" },
      });
    });

    it("should refuse an empty sync sequence", () => {
      expect(() => parseFraming({ type: "sync", sync: "" })).toThrow();
    });
  });
});
