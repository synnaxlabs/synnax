// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type record } from "@synnaxlabs/x";

/**
 * @returns the top-level fields of next whose values differ from prev, compared deeply.
 * A field that next removes maps to undefined, so merging the result over prev with a
 * spread gives next.
 */
export const delta = (prev: record.Unknown, next: record.Unknown): record.Unknown => {
  const changed: record.Unknown = {};
  for (const key of Object.keys(next))
    if (!(key in prev) || !deep.equal(prev[key], next[key])) changed[key] = next[key];
  for (const key of Object.keys(prev)) if (!(key in next)) changed[key] = undefined;
  return changed;
};
