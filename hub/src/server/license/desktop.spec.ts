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
  expiryFrom,
  hashSecret,
  type Machine,
  mintSecret,
  superseded,
  TERM_MS,
} from "@/server/license/desktop";
import {
  activationOf,
  HASH_A,
  HASH_B,
  HASH_C,
  LICENSE,
  NOW,
} from "@/server/license/testutil";

describe("desktop", () => {
  describe("mintSecret", () => {
    it("should mint a URL-safe secret that differs each time", () => {
      const a = mintSecret();
      const b = mintSecret();
      expect(a).toMatch(/^[A-Za-z0-9_-]{43}$/);
      expect(a).not.toBe(b);
    });
  });
  describe("hashSecret", () => {
    it("should hash the same secret to the same digest", () => {
      expect(hashSecret("s")).toBe(hashSecret("s"));
      expect(hashSecret("s")).toMatch(/^[0-9a-f]{64}$/);
      expect(hashSecret("s")).not.toBe(hashSecret("t"));
    });
  });
  describe("expiryFrom", () => {
    it("should put the expiry one term after now", () => {
      expect(expiryFrom(NOW).getTime() - NOW.getTime()).toBe(TERM_MS);
    });
  });
  describe("superseded", () => {
    const machineOf = (fingerprint: string[]): Machine => ({
      activation: activationOf(fingerprint),
      license: LICENSE,
    });
    it("should pick the machines that share any hash", () => {
      const same = machineOf([HASH_A, HASH_B]);
      const other = machineOf([HASH_C]);
      expect(superseded([same, other], [HASH_B])).toEqual([same]);
    });
    it("should pick nothing for a new machine", () => {
      expect(superseded([machineOf([HASH_A])], [HASH_C])).toEqual([]);
    });
  });
});
