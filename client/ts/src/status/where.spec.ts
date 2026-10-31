// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { statusZ, whereZ } from "@/status/types.gen";
import { type Lines, matches, serverComputed } from "@/status/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

const lines: Lines = {
  labels: (s, keys) => s.labels?.some((l) => keys.includes(l.key)) ?? false,
};

describe("Where", () => {
  itMatchesVectors(
    readVectors("statuses", statusZ(), whereZ),
    (s, where) => matches(lines, s, where),
    (where) => serverComputed(lines, where),
  );
});
