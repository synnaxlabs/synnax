// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { payloadZ, whereZ } from "@/ranger/types.gen";
import { createLines } from "@/ranger/where";
import { matches, serverComputed } from "@/ranger/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

const lines = createLines(
  (r, keys) => r.labels?.some((l) => keys.includes(l.key)) ?? false,
);

describe("Where", () => {
  itMatchesVectors(
    readVectors("ranges", payloadZ, whereZ),
    (r, where) => matches(lines, r, where),
    (where) => serverComputed(lines, where),
  );
});
