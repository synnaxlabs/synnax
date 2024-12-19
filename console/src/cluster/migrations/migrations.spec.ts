// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { migrateSlice, ZERO_SLICE_STATE } from "@/cluster/migrations";
import * as v0 from "@/cluster/migrations/v0";
import * as v1 from "@/cluster/migrations/v1";
import * as v2 from "@/cluster/migrations/v2";

const STATES = [v0.ZERO_SLICE_STATE, v1.ZERO_SLICE_STATE, v2.ZERO_SLICE_STATE];

describe("migrations", () =>
  STATES.forEach((state) =>
    it(`should migrate slice from ${state.version} to latest`, () =>
      expect(migrateSlice(state)).toEqual(ZERO_SLICE_STATE)),
  ));
