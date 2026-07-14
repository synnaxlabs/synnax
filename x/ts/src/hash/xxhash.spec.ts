// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { hash } from "@/hash";
import { json } from "@/json";

describe("xxHash64", () => {
  // Reference vectors from the xxHash specification (seed 0).
  it("should hash the empty input", () => {
    expect(hash.xxHash64("")).toEqual("ef46db3751d8e999");
  });

  it("should hash inputs shorter than 32 bytes", () => {
    expect(hash.xxHash64("a")).toEqual(hash.xxHash64(new Uint8Array([0x61])));
  });

  it("should hash inputs of every length mod 32", () => {
    // Exercises the stripe, 8-byte, 4-byte, and tail paths together.
    const long = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQ";
    for (let i = 0; i < long.length; i++)
      expect(hash.xxHash64(long.slice(0, i))).toHaveLength(16);
  });

  // Golden vectors shared with Go (core task.HashConfig) and C++ (x/cpp/hash):
  // identical configs must hash identically in every language.
  describe("cross-language golden vectors", () => {
    it("should hash the empty config", () => {
      expect(hash.xxHash64(json.canonicalString({}))).toEqual("2e1472b57af294d1");
    });

    it("should hash a flat config", () => {
      expect(
        hash.xxHash64(
          json.canonicalString({ rate: 50, port: 8080, host: "localhost" }),
        ),
      ).toEqual("2de66015b3bdded8");
    });

    it("should hash a nested config", () => {
      expect(
        hash.xxHash64(
          json.canonicalString({
            enabled: true,
            channels: [{ key: 12, name: 'ch"1"', scale: 0.001 }],
            notes: "héllo⚡",
          }),
        ),
      ).toEqual("811ef1fc462a59f2");
    });
  });
});
