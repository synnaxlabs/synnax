// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The offset of each byte of a canonical UUID in its text, from the last byte to the
// first.
const BYTE_OFFSETS = [34, 32, 30, 28, 26, 24, 21, 19, 16, 14, 11, 9, 6, 4, 2, 0];

/**
 * Orders canonical lower case UUIDs as the Core stores them: byte by byte, from the
 * last byte to the first.
 *
 * @returns a negative number if a orders before b, a positive number if after, and 0
 * if they are equal.
 */
export const uuids = (a: string, b: string): number => {
  for (const offset of BYTE_OFFSETS)
    for (let j = offset; j < offset + 2; j++) {
      const diff = a.charCodeAt(j) - b.charCodeAt(j);
      if (diff !== 0) return Math.sign(diff);
    }
  return 0;
};
