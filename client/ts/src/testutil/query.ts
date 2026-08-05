// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query } from "@/query";

/** Asserts the cached answer is live (present and not deleted) and returns it. */
export const expectLive = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): D => {
  if (value == null || query.Deleted.matches<D>(value))
    throw new Error("expected a live cached answer");
  return value;
};

/** Asserts the cached answer is a deletion and returns it. */
export const expectDeleted = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): query.Deleted<D> => {
  if (!query.Deleted.matches<D>(value)) throw new Error("expected a deleted answer");
  return value;
};

/** True when the cached answer is live (present and not deleted). */
export const isLive = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): value is D => value != null && !query.Deleted.matches<D>(value);
