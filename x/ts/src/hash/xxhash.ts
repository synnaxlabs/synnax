// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const MASK = 0xffffffffffffffffn;
const P1 = 0x9e3779b185ebca87n;
const P2 = 0xc2b2ae3d27d4eb4fn;
const P3 = 0x165667b19e3779f9n;
const P4 = 0x85ebca77c2b2ae63n;
const P5 = 0x27d4eb2f165667c5n;

const rotl = (v: bigint, r: bigint): bigint => ((v << r) & MASK) | (v >> (64n - r));

const round = (acc: bigint, lane: bigint): bigint =>
  (rotl((acc + ((lane * P2) & MASK)) & MASK, 31n) * P1) & MASK;

const mergeRound = (h: bigint, acc: bigint): bigint =>
  ((h ^ round(0n, acc)) * P1 + P4) & MASK;

const readU64 = (b: Uint8Array, i: number): bigint => {
  let v = 0n;
  for (let j = 7; j >= 0; j--) v = (v << 8n) | BigInt(b[i + j]);
  return v;
};

const readU32 = (b: Uint8Array, i: number): bigint =>
  BigInt(b[i] | (b[i + 1] << 8) | (b[i + 2] << 16)) | (BigInt(b[i + 3]) << 24n);

/**
 * Computes the 64-bit xxHash (XXH64, seed 0) of the input. Strings are hashed
 * over their UTF-8 bytes.
 * @param input - The string or bytes to hash.
 * @returns The hash as a 16-character zero-padded lowercase hex string.
 */
export const xxHash64 = (input: string | Uint8Array): string => {
  const b = typeof input === "string" ? new TextEncoder().encode(input) : input;
  const len = b.length;
  let h: bigint;
  let i = 0;
  if (len >= 32) {
    let a1 = (P1 + P2) & MASK;
    let a2 = P2;
    let a3 = 0n;
    let a4 = (0n - P1) & MASK;
    for (; i + 32 <= len; i += 32) {
      a1 = round(a1, readU64(b, i));
      a2 = round(a2, readU64(b, i + 8));
      a3 = round(a3, readU64(b, i + 16));
      a4 = round(a4, readU64(b, i + 24));
    }
    h = (rotl(a1, 1n) + rotl(a2, 7n) + rotl(a3, 12n) + rotl(a4, 18n)) & MASK;
    h = mergeRound(h, a1);
    h = mergeRound(h, a2);
    h = mergeRound(h, a3);
    h = mergeRound(h, a4);
  } else h = P5;
  h = (h + BigInt(len)) & MASK;
  for (; i + 8 <= len; i += 8)
    h = (rotl(h ^ round(0n, readU64(b, i)), 27n) * P1 + P4) & MASK;
  if (i + 4 <= len) {
    h = (rotl(h ^ ((readU32(b, i) * P1) & MASK), 23n) * P2 + P3) & MASK;
    i += 4;
  }
  for (; i < len; i++) h = (rotl(h ^ ((BigInt(b[i]) * P5) & MASK), 11n) * P1) & MASK;
  h ^= h >> 33n;
  h = (h * P2) & MASK;
  h ^= h >> 29n;
  h = (h * P3) & MASK;
  h ^= h >> 32n;
  return h.toString(16).padStart(16, "0");
};
