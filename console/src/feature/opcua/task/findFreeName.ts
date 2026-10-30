// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

/**
 * @returns base, or base with the lowest numeric suffix, such that no channel and no
 * name in reserved holds it or its index name. Adds both names to reserved.
 */
export const findFreeName = async (
  client: Synnax,
  base: string,
  reserved: Set<string>,
): Promise<string> => {
  for (let n = 1; ; n++) {
    const name = n === 1 ? base : `${base}_${n}`;
    const index = `${name}_time`;
    if (reserved.has(name) || reserved.has(index)) continue;
    const taken = await client.channels.retrieve([name, index]);
    if (taken.length > 0) continue;
    reserved.add(name).add(index);
    return name;
  }
};
