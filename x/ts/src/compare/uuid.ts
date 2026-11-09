// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The offset of each byte of a canonical UUID in its text.
const BYTE_OFFSETS = [0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34];

/**
 * Orders canonical lower case UUIDs as the Core stores them: byte by byte, from the
 * last byte to the first.
 *
 * @returns a negative number if a orders before b, a positive number if after, and 0
 * if they are equal.
 */
export const uuids = (a: string, b: string): number => {
  for (let i = BYTE_OFFSETS.length - 1; i >= 0; i--)
    for (let j = BYTE_OFFSETS[i]; j < BYTE_OFFSETS[i] + 2; j++) {
      const diff = a.charCodeAt(j) - b.charCodeAt(j);
      if (diff !== 0) return Math.sign(diff);
    }
  return 0;
};
