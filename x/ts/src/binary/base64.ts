// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * The alphabet of a base64 string. "base64url" (RFC 4648 §5) swaps "+" and "/" for "-"
 * and "_" and has no padding, so the string is safe to embed in a URL.
 */
export type Base64Alphabet = "base64" | "base64url";

export interface Base64Options {
  alphabet?: Base64Alphabet;
}

// btoa takes a binary string, and spreading a large array into fromCharCode overflows
// the stack, so the bytes convert in chunks.
const CHUNK_SIZE = 0x8000;

/** @returns the bytes encoded as a base64 string in the given alphabet. */
export const encodeBase64 = (
  bytes: Uint8Array,
  { alphabet = "base64" }: Base64Options = {},
): string => {
  let binary = "";
  for (let i = 0; i < bytes.length; i += CHUNK_SIZE)
    binary += String.fromCharCode(...bytes.subarray(i, i + CHUNK_SIZE));
  const encoded = btoa(binary);
  if (alphabet === "base64") return encoded;
  return encoded.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
};

/**
 * @returns the bytes of a base64 string in the given alphabet. Padding is optional.
 * @throws {DOMException} when the string is not valid base64.
 */
export const decodeBase64 = (
  encoded: string,
  { alphabet = "base64" }: Base64Options = {},
): Uint8Array<ArrayBuffer> => {
  const standard =
    alphabet === "base64" ? encoded : encoded.replace(/-/g, "+").replace(/_/g, "/");
  return Uint8Array.from(atob(standard), (char) => char.charCodeAt(0));
};
