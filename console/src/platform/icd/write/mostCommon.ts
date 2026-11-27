// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** @returns the item with the largest count, the first of a tie, or null for none. */
export const mostCommon = <T>(items: T[], countOf: (item: T) => number): T | null =>
  items.reduce<T | null>(
    (top, item) => (top == null || countOf(item) > countOf(top) ? item : top),
    null,
  );
