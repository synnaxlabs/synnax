// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resolve } from "@/deep/resolve";

/**
 * Removes the value at a dot-separated path, found as {@link resolve} finds it: a key
 * from an object or an entry from an array. Does nothing when nothing holds the path.
 */
export const remove = <V>(obj: V, path: string): void => {
  const found = resolve(obj, path);
  if (found == null) return;
  if (Array.isArray(found.parent)) found.parent.splice(found.key as number, 1);
  else delete found.parent[found.key];
};
