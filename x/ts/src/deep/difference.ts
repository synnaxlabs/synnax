// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

type PathValueTuple = [any, any];

export const difference = (
  obj1: Record<string, any>,
  obj2: Record<string, any>,
  path: string = "",
): Record<string, PathValueTuple> => {
  const diffMap: Record<string, PathValueTuple> = {};

  const compare = (a: unknown, b: unknown, currentPath: string): void => {
    if (typeof a !== typeof b || a === null || b === null) {
      diffMap[currentPath] = [a, b];
      return;
    }

    if (typeof a === "object" && typeof b === "object")
      if (Array.isArray(a) && Array.isArray(b)) {
        if (a.length !== b.length) {
          diffMap[currentPath] = [a, b];
          return;
        }
        for (let i = 0; i < a.length; i++) compare(a[i], b[i], `${currentPath}[${i}]`);
      } else {
        const keys = new Set([...Object.keys(a as {}), ...Object.keys(b as {})]);
        keys.forEach((key) => {
          compare(
            a[key as keyof typeof a],
            b[key as keyof typeof b],
            currentPath !== "" ? `${currentPath}.${key}` : key,
          );
        });
      }
    else if (a !== b) diffMap[currentPath] = [a, b];
  };

  compare(obj1, obj2, path);
  return diffMap;
};
