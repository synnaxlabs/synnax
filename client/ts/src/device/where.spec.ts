// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { deviceZ, whereZ } from "@/device/types.gen";
import { matches, serverComputed } from "@/device/where.gen";
import { itMatchesVectors, readVectors } from "@/testutil/vectors";

describe("Where", () => {
  itMatchesVectors(readVectors("devices", deviceZ(), whereZ), matches, serverComputed);
});
