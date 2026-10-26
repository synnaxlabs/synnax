// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { payloadZ, whereZ } from "@/channel/types.gen";
import { LINES } from "@/channel/where";
import { matches, serverComputed } from "@/channel/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  itMatchesVectors(
    readVectors("channels", payloadZ, whereZ),
    (c, where) => matches(LINES, c, where),
    (where) => serverComputed(LINES, where),
  );
});
