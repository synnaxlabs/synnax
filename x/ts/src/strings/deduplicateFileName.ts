// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const deduplicateFileName = (
  name: string,
  existingNames: Set<string>,
): string => {
  if (!existingNames.has(name)) return name;
  let baseName = name;
  let i = 1;
  let currentName = name;
  while (existingNames.has(currentName)) {
    const match = currentName.match(filenameEndingRegex);
    if (match) {
      baseName = currentName.slice(0, match.index).trim();
      i = parseInt(match[1]) + 1;
    } else {
      baseName = currentName;
      i = 1;
    }
    currentName = `${baseName} (${i})`;
  }
  return currentName;
};

const filenameEndingRegex = /\((\d+)\)$/;
