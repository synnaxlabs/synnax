// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { filter } from "@synnaxlabs/x";

import { isCalculated } from "@/channel/payload";
import { type Key, type Payload } from "@/channel/types.gen";
import { type Lines } from "@/channel/where.gen";
import { group } from "@/group";

/**
 * @returns the typed lines of a channel Where. The internal line checks the keys that
 * builtIns returns, and only the Core checks it while builtIns returns null.
 */
export const createLines = (builtIns: () => Set<Key> | null): Lines => ({
  ...group.MEMBER_LINES,
  get internal() {
    const keys = builtIns();
    if (keys == null) return null;
    return (ch: Payload, value: filter.Bool) =>
      filter.matchesSet(value, keys.has(ch.key));
  },
  virtual: (ch, value) => (ch.virtual && !isCalculated(ch)) === value,
});
