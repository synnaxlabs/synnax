// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A key that moves a cursor through a vertical run of items. */
export type MoveKey = "ArrowUp" | "ArrowDown" | "Home" | "End";

const MOVE_KEYS: ReadonlySet<string> = new Set<MoveKey>([
  "ArrowUp",
  "ArrowDown",
  "Home",
  "End",
]);

export const isMoveKey = (key: string | null): key is MoveKey =>
  key != null && MOVE_KEYS.has(key);

/**
 * Returns the index that a key moves a cursor to from index at, where -1 is no cursor,
 * or null when the cursor stays. A held key stops at the ends, and a new press wraps
 * around.
 */
export const moveIndex = (
  key: MoveKey,
  at: number,
  count: number,
  held: boolean,
): number | null => {
  const last = count - 1;
  if (key === "Home") return 0;
  if (key === "End") return last;
  if (key === "ArrowDown") {
    if (at === -1) return 0;
    if (at < last) return at + 1;
    return held ? null : 0;
  }
  if (at === -1) return last;
  if (at > 0) return at - 1;
  return held ? null : last;
};
