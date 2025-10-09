// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { defaultGetter, SEPARATOR } from "@/deep/path";
import { type record } from "@/record";

export const remove = <V>(obj: V, path: string): void => {
  const parts = path.split(SEPARATOR);
  let result: record.Unknown = obj as record.Unknown;

  let i = 0;
  while (i < parts.length) {
    if (i === parts.length - 1) {
      // Last part - perform deletion
      const lastPart = parts[i];

      if (Array.isArray(result)) {
        const index = parseInt(lastPart);
        if (!isNaN(index) && index < result.length) {
          result.splice(index, 1);
          return;
        }

        const first = result[0];
        if (typeof first === "object" && "key" in first) {
          const objIndex = result.findIndex((o) => o.key === lastPart);
          if (objIndex !== -1) {
            result.splice(objIndex, 1);
            return;
          }
        }
        return;
      }

      delete result[lastPart];
      return;
    }

    // Not the last part - navigate deeper
    let found = false;

    // First try to match with keyed items in arrays
    if (Array.isArray(result) && result.length > 0) {
      const first = result[0];
      if (typeof first === "object" && "key" in first)
        // Try to find an item with a matching key (considering periods in keys)
        for (let j = parts.length - i; j >= 1; j--) {
          const candidateKey = parts.slice(i, i + j).join(SEPARATOR);
          const item = result.find((o) => o.key === candidateKey);
          if (item != null) {
            if (i + j === parts.length) {
              // This is the item to remove
              const objIndex = result.findIndex((o) => o.key === candidateKey);
              if (objIndex !== -1) result.splice(objIndex, 1);

              return;
            }
            result = item as record.Unknown;
            i += j;
            found = true;
            break;
          }
        }
    }

    if (!found)
      // Try to match properties with periods in objects
      for (let j = parts.length - i; j >= 1; j--) {
        const candidateKey = parts.slice(i, i + j).join(SEPARATOR);
        if (
          !Array.isArray(result) &&
          typeof result === "object" &&
          result !== null &&
          candidateKey in result
        ) {
          if (i + j === parts.length) {
            // This is the property to remove
            delete result[candidateKey];
            return;
          }
          result = result[candidateKey] as record.Unknown;
          i += j;
          found = true;
          break;
        }
      }

    if (!found) {
      // Try normal property access
      const next = defaultGetter(result, parts[i]);
      if (next == null) return;
      result = next as record.Unknown;
      i++;
    }
  }
};
