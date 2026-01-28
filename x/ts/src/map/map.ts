// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const getOrSetDefault = <K, V>(map: Map<K, V>, key: K, defaultValue: V): V => {
  const value = map.get(key);
  if (value === undefined) {
    map.set(key, defaultValue);
    return defaultValue;
  }
  return value;
};
