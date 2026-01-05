// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const copy = <T>(obj: T): T => {
  try {
    return structuredClone(obj);
  } catch (_) {
    console.warn(
      "Failed to deep copy object, falling back to JSON.parse(JSON.stringify)",
      obj,
    );
    console.trace();
    return JSON.parse(JSON.stringify(obj));
  }
};
