// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { uuid } from "@/uuid";

describe("UUID", () => {
  describe("create", () => {
    it("should generate a valid UUID v4", () => {
      const uid = uuid.create();
      expect(uid).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
      );
    });

    it("should generate unique UUIDs", () => {
      const uuids = new Set();
      for (let i = 0; i < 1000; i++) {
        const uid = uuid.create();
        expect(uuids.has(uid)).toBe(false);
        uuids.add(uid);
      }
    });
  });

  describe("fromName", () => {
    it("should derive the RFC 9562 version 5 UUID", () => {
      // uuid5(NAMESPACE_DNS, "www.example.com") from the RFC test vectors.
      expect(
        uuid.fromName("www.example.com", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
      ).toEqual("2ed6657d-e927-568b-95e1-2665a8aea6a2");
    });

    it("should derive the same UUID for the same name", () => {
      const ns = uuid.create();
      expect(uuid.fromName("a", ns)).toEqual(uuid.fromName("a", ns));
    });

    it("should derive different UUIDs for different names", () => {
      const ns = uuid.create();
      expect(uuid.fromName("a", ns)).not.toEqual(uuid.fromName("b", ns));
    });
  });

  describe("parse", () => {
    it("should parse a valid UUID from bytes", () => {
      const bytes = new Uint8Array([
        0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14,
        0x17, 0x40, 0x00,
      ]);
      const uid = uuid.parse(bytes);
      expect(uid).toBe("123e4567-e89b-12d3-a456-426614174000");
    });

    it("should handle zero bytes", () => {
      const bytes = new Uint8Array(16);
      const uid = uuid.parse(bytes);
      expect(uid).toBe("00000000-0000-0000-0000-000000000000");
    });

    it("should handle all ones", () => {
      const bytes = new Uint8Array(16).fill(0xff);
      const uid = uuid.parse(bytes);
      expect(uid).toBe("ffffffff-ffff-ffff-ffff-ffffffffffff");
    });

    it("should handle random bytes", () => {
      const bytes = new Uint8Array([
        0x7f, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14,
        0x17, 0x40, 0x00,
      ]);
      const uid = uuid.parse(bytes);
      expect(uid).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
      );
    });

    it("should throw error for invalid byte length", () => {
      const bytes = new Uint8Array(15); // Invalid length
      expect(() => uuid.parse(bytes)).toThrow();
    });

    it("should maintain byte order", () => {
      // Test with a specific pattern to verify byte order
      const bytes = new Uint8Array([
        0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd,
        0xee, 0xff, 0x00,
      ]);
      const uid = uuid.parse(bytes);
      expect(uid).toBe("11223344-5566-7788-99aa-bbccddeeff00");
    });
  });
});
