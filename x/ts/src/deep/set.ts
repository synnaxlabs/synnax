// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { defaultGetter, findBestKey, getIndex, SEPARATOR } from "@/deep/path";
import { type record } from "@/record";

export const set = <V>(obj: V, path: string, value: unknown): void => {
  const parts = path.split(SEPARATOR);
  let result: record.Unknown = obj as record.Unknown;

  let i = 0;
  while (i < parts.length - 1) {
    const best = findBestKey(result, parts.slice(i, parts.length - 1));
    let part: string;
    if (best != null) [part, i] = [best[0], i + best[1]];
    else {
      part = parts[i];
      i++;
    }
    let v = defaultGetter(result, part);
    if (v == null) {
      const nextPart = parts[i];
      let idx = getIndex(nextPart);
      if (idx == null && nextPart.startsWith("-")) {
        const negIndex = getIndex(nextPart.substring(1));
        if (negIndex != null) idx = 0;
      }
      v = idx != null ? [] : {};
      result[part] = v;
    }
    result = v as record.Unknown;
  }
  try {
    const lastPart = parts[parts.length - 1];

    // Handle arrays specially
    if (Array.isArray(result)) {
      let index = getIndex(lastPart);
      if (index == null) {
        // Check for negative index
        if (lastPart.startsWith("-")) {
          const negIndex = getIndex(lastPart.substring(1));
          if (negIndex != null) index = result.length - negIndex;
        }

        // If still no valid index, try keyed array logic
        if (index == null) {
          if (result.length === 0) {
            // For empty arrays, try to set at numeric index
            const idx = getIndex(lastPart);
            if (idx != null) {
              result[idx] = value;
              return;
            }
          }
          // Check if it's a keyed array
          const first = result[0];
          if (typeof first === "object" && "key" in first) {
            const objIndex = result.findIndex((o) => o.key === lastPart);
            if (objIndex !== -1) {
              result[objIndex] = value;
              return;
            }
          }
          // Can't find a valid way to set on this array
          return;
        }
      }
      // Set at the calculated index
      result[index] = value;
      return;
    }

    // Handle objects
    const best = findBestKey(result, [lastPart]);
    if (best != null) {
      result[best[0]] = value;
      return;
    }
    result[lastPart] = value;
  } catch (e) {
    console.error("failed to set value", value, "at path", path, "on object", obj);
    throw e;
  }
};
