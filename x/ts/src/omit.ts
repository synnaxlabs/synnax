// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const omit = <T, K extends keyof T>(obj: T, ...keys: K[]): Omit<T, K> => {
  const result = { ...obj };
  for (const key of keys) delete result[key];
  return result;
};
