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

    const SYNC = { type: "sync", sync: "qg==" };
    const BYTE_ORDER = 'Invalid option: expected one of "little_endian"|"big_endian"';
    const ALGORITHM =
      'Invalid option: expected one of "crc16_ccitt_false"|"crc16_modbus"|"crc32"';
    const REFUSALS: [string, unknown, string[], string][] = [
      [
        "an empty sync sequence",
        { type: "sync", sync: "" },
        ["sync"],
        "must be at least 1 bytes long",
      ],
      [
        "an unknown length size",
        { ...SYNC, lengthSize: "three" },
        ["lengthSize"],
        'Invalid option: expected one of "one"|"two"|"four"',
      ],
      [
        "a max length of 0",
        { ...SYNC, maxLength: 0 },
        ["maxLength"],
        "Too small: expected number to be >=1",
      ],
      [
        "a max length of 65537",
        { ...SYNC, maxLength: 65537 },
        ["maxLength"],
        "Too big: expected number to be <=65536",
      ],
      [
        "an unknown byte order",
        { ...SYNC, byteOrder: "middle_endian" },
        ["byteOrder"],
        BYTE_ORDER,
      ],
      [
        "an unknown checksum byte order",
        { ...SYNC, checksum: { algorithm: "crc32", byteOrder: "middle_endian" } },
        ["checksum", "byteOrder"],
        BYTE_ORDER,
      ],
      [
        "a checksum with no algorithm",
        { ...SYNC, checksum: {} },
        ["checksum", "algorithm"],
        ALGORITHM,
      ],
      [
        "an unknown checksum algorithm",
        { ...SYNC, checksum: { algorithm: "crc8" } },
        ["checksum", "algorithm"],
        ALGORITHM,
      ],
      [
        "a fixed length of 0",
        { type: "fixed", length: 0 },
        ["length"],
        "Too small: expected number to be >=1",
      ],
      [
        "a fixed length of 65537",
        { type: "fixed", length: 65537 },
        ["length"],
        "Too big: expected number to be <=65536",
      ],
      [
        "an empty delimiter",
        { type: "delimited", delimiter: "" },
        ["delimiter"],
        "Too small: expected string to have >=1 characters",
      ],
      [
        "an unknown framing type",
        { type: "bogus" },
        ["type"],
        "Invalid discriminator value. Expected 'delimited' | 'fixed' | 'sync' | 'cobs' | 'slip'",
      ],
    ];
    it.each(REFUSALS)(
      "should refuse %s with the path of its field",
      (_, framing, path, message) => {
        const result = icd.icdZ.safeParse({ ...ICD, framing });
        expect(result.error?.issues).toMatchObject([
          { path: ["framing", ...path], message },
        ]);
      },
    );
  });
});
