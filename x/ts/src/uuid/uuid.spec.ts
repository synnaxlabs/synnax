// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { validate } from "uuid";
import { describe, expect, it } from "vitest";

import { uuid } from "@/uuid";

describe("UUID", () => {
  describe("create", () => {
    it("should create a valid UUID", () => {
      const uid = uuid.create();
      expect(validate(uid)).toEqual(true);
      expect(uid).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
      );
    });
  });

  describe("uuidZ", () => {
    it("should validate a valid UUID string", () => {
      const uid = uuid.create();
      const result = uuid.uuidZ.safeParse(uid);
      expect(result.success).toEqual(true);
      if (result.success) expect(result.data).toEqual(uid);
    });

    it("should reject an invalid UUID string", () => {
      const result = uuid.uuidZ.safeParse("not-a-uuid");
      expect(result.success).toEqual(false);
    });

    it("should parse a UUID from a Uint8Array", () => {
      const bytes = new Uint8Array([
        0x6e, 0x04, 0x57, 0xf4, 0x45, 0x75, 0x4a, 0xdf, 0xb5, 0x9d, 0x83, 0x31, 0x14,
        0x59, 0xa7, 0xa5,
      ]);

      const result = uuid.uuidZ.safeParse(bytes);
      expect(result.success).toEqual(true);

      if (result.success) {
        expect(result.data).toEqual("6e0457f4-4575-4adf-b59d-83311459a7a5");
        expect(validate(result.data)).toEqual(true);
      }
    });

    it("should handle the validation of UUIDs from different sources", () => {
      const uid = uuid.create();

      const stringResult = uuid.uuidZ.safeParse(uid);
      expect(stringResult.success).toEqual(true);

      const bytes = new Uint8Array(16);
      for (let i = 0; i < uid.replace(/-/g, "").length; i += 2)
        bytes[i / 2] = parseInt(uid.replace(/-/g, "").substring(i, i + 2), 16);

      const bytesResult = uuid.uuidZ.safeParse(bytes);
      expect(bytesResult.success).toEqual(true);

      if (stringResult.success && bytesResult.success)
        expect(bytesResult.data).toEqual(uid);
    });
  });
});
