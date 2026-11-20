// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { status } from "@/status";
import { LINES } from "@/status/where";
import { matches, serverComputed } from "@/status/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  itMatchesVectors(
    readVectors("statuses", status.statusZ(), status.whereZ),
    (s, where) => matches(LINES, s, where),
    (where) => serverComputed(LINES, where),
  );
});
