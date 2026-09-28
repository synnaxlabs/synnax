// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { decide } from "@/server/license/activate";
import {
  activationOf,
  HASH_A,
  HASH_B,
  HASH_C,
  LICENSE,
  NOW,
} from "@/server/license/testutil";

describe("activate.decide", () => {
  it("should grant a seat when one is free", () => {
    expect(
      decide({ license: LICENSE, activations: [], fingerprint: [HASH_A], now: NOW }),
    ).toEqual({ ok: true });
  });

  it("should keep the seat of a machine that shares any hash", () => {
    const held = activationOf([HASH_A, HASH_B]);
    const decision = decide({
      license: LICENSE,
      activations: [held],
      fingerprint: [HASH_B, HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true, existing: held });
  });

  it("should refuse a new machine when every seat is taken", () => {
    const decision = decide({
      license: LICENSE,
      activations: [activationOf([HASH_A]), activationOf([HASH_B])],
      fingerprint: [HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: false, reason: "no_seats" });
  });

  it("should not count a released seat", () => {
    const decision = decide({
      license: LICENSE,
      activations: [
        activationOf([HASH_A]),
        activationOf([HASH_B], { releasedAt: NOW }),
      ],
      fingerprint: [HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true });
  });

  it("should let a released machine reactivate as a new seat", () => {
    const decision = decide({
      license: LICENSE,
      activations: [activationOf([HASH_A], { releasedAt: NOW })],
      fingerprint: [HASH_A],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true });
  });

  it("should refuse a revoked license", () => {
    expect(
      decide({
        license: { ...LICENSE, revokedAt: NOW },
        activations: [],
        fingerprint: [HASH_A],
        now: NOW,
      }),
    ).toEqual({ ok: false, reason: "revoked" });
  });

  it("should refuse an expired subscription without a fallback", () => {
    expect(
      decide({
        license: { ...LICENSE, expiresAt: new Date(NOW.getTime() - 1) },
        activations: [],
        fingerprint: [HASH_A],
        now: NOW,
      }),
    ).toEqual({ ok: false, reason: "expired" });
  });

  it("should still activate an expired subscription that has a fallback", () => {
    expect(
      decide({
        license: {
          ...LICENSE,
          expiresAt: new Date(NOW.getTime() - 1),
          maxVersion: "0.60",
        },
        activations: [],
        fingerprint: [HASH_A],
        now: NOW,
      }),
    ).toEqual({ ok: true });
  });
});
