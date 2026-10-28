// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const PEM_ARMOR = /-----(BEGIN|END) CERTIFICATE-----|\s/g;

/** @returns the DER bytes of a PEM certificate. */
export const toDER = (pem: string): Uint8Array<ArrayBuffer> =>
  Uint8Array.from(atob(pem.replace(PEM_ARMOR, "")), (c) => c.charCodeAt(0));

const rotl = (x: number, n: number): number => (x << n) | (x >>> (32 - n));

// Web Crypto exists only in a secure context, so a Console served over plain HTTP
// computes the digest itself.
const sha1 = (data: Uint8Array): Uint8Array => {
  const padded = new Uint8Array(Math.ceil((data.length + 9) / 64) * 64);
  padded.set(data);
  padded[data.length] = 0x80;
  const view = new DataView(padded.buffer);
  view.setUint32(padded.length - 8, Math.floor(data.length / 0x20000000));
  view.setUint32(padded.length - 4, (data.length * 8) >>> 0);
  const h = [0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0];
  const w = new Uint32Array(80);
  for (let block = 0; block < padded.length; block += 64) {
    for (let i = 0; i < 16; i++) w[i] = view.getUint32(block + i * 4);
    for (let i = 16; i < 80; i++)
      w[i] = rotl(w[i - 3] ^ w[i - 8] ^ w[i - 14] ^ w[i - 16], 1);
    let [a, b, c, d, e] = h;
    for (let i = 0; i < 80; i++) {
      let f: number;
      let k: number;
      if (i < 20) {
        f = (b & c) | (~b & d);
        k = 0x5a827999;
      } else if (i < 40) {
        f = b ^ c ^ d;
        k = 0x6ed9eba1;
      } else if (i < 60) {
        f = (b & c) | (b & d) | (c & d);
        k = 0x8f1bbcdc;
      } else {
        f = b ^ c ^ d;
        k = 0xca62c1d6;
      }
      const t = (rotl(a, 5) + f + e + k + w[i]) >>> 0;
      e = d;
      d = c;
      c = rotl(b, 30) >>> 0;
      b = a;
      a = t;
    }
    h[0] = (h[0] + a) >>> 0;
    h[1] = (h[1] + b) >>> 0;
    h[2] = (h[2] + c) >>> 0;
    h[3] = (h[3] + d) >>> 0;
    h[4] = (h[4] + e) >>> 0;
  }
  const digest = new Uint8Array(20);
  const out = new DataView(digest.buffer);
  h.forEach((v, i) => out.setUint32(i * 4, v));
  return digest;
};

/**
 * @returns the SHA-1 thumbprint of a PEM certificate as uppercase hex, the form that
 * the Driver reports.
 * @throws if the PEM body is not valid base64.
 */
export const thumbprint = (pem: string): string =>
  Array.from(sha1(toDER(pem)), (b) => b.toString(16).padStart(2, "0"))
    .join("")
    .toUpperCase();
