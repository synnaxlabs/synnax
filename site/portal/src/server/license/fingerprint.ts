// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The browser imports this file, so it must not import anything that reaches the
// database.

const HASH = /^[0-9a-f]{64}$/;

/**
 * parse reads a machine's fingerprint: the hashes a Core prints, as a comma, space, or
 * newline separated list. Returns the unique hashes in order, or throws when the list
 * is empty or holds anything that is not a lowercase SHA-256 hex digest.
 */
export const parse = (text: string): string[] => {
  const hashes = text
    .split(/[\s,]+/)
    .map((h) => h.trim().toLowerCase())
    .filter((h) => h.length > 0);
  if (hashes.length === 0) throw new Error("Enter the machine's fingerprint");
  const bad = hashes.find((h) => !HASH.test(h));
  if (bad != null) throw new Error(`"${bad}" is not part of a fingerprint`);
  return [...new Set(hashes)];
};

/**
 * read parses a fingerprint pasted on its own or inside the activation link a Core
 * prints. Throws as {@link parse} does.
 */
export const read = (text: string): string[] => {
  const trimmed = text.trim();
  if (!URL.canParse(trimmed)) return parse(trimmed);
  return parse(new URL(trimmed).searchParams.get("fingerprint") ?? "");
};
