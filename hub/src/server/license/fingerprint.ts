// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const HASH = /^[0-9a-f]{64}$/;

/**
 * parse reads the host hashes a Core prints, pasted as a comma, space, or newline
 * separated list. Returns the unique hashes in order, or throws when the list is
 * empty or holds anything that is not a lowercase SHA-256 hex digest.
 */
export const parse = (text: string): string[] => {
  const hashes = text
    .split(/[\s,]+/)
    .map((h) => h.trim().toLowerCase())
    .filter((h) => h.length > 0);
  if (hashes.length === 0) throw new Error("Paste at least one host hash");
  const bad = hashes.find((h) => !HASH.test(h));
  if (bad != null) throw new Error(`"${bad}" is not a host hash`);
  return [...new Set(hashes)];
};
