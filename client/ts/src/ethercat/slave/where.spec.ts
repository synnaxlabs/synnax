// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe } from "vitest";

import { ethercat } from "@/ethercat";
import { matches, serverComputed } from "@/ethercat/slave/where.gen";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  vectors.itMatches(
    vectors.read("ethercatSlaves", ethercat.slave.slaveZ, ethercat.slave.whereZ),
    matches,
    serverComputed,
  );
});
