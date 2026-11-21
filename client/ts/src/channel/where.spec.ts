// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { channel } from "@/channel";
import { createLines } from "@/channel/where";
import { matches, serverComputed } from "@/channel/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  // The internal records of the vectors are the built-in channels.
  const recordZ = channel.payloadZ.extend({ internal: z.boolean().default(false) });
  const vectors = readVectors("channels", recordZ, channel.whereZ);
  const builtIns = new Set(
    vectors.records.filter(([, ch]) => ch.internal).map(([, ch]) => ch.key),
  );
  const lines = createLines(() => builtIns);
  itMatchesVectors(
    vectors,
    (c, where) => matches(lines, c, where),
    (where) => serverComputed(lines, where),
  );

  it("should leave the internal line to the Core before the built-in keys arrive", () => {
    const unknown = createLines(() => null);
    expect(serverComputed(unknown, { internal: { eq: false } })).toBe(true);
    expect(serverComputed(unknown, { virtual: true })).toBe(false);
  });
});
