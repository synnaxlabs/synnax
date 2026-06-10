// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { migrateLegacySlice, ZERO_SLICE_STATE } from "@/project/types";

describe("migrateLegacySlice", () => {
  const active = { key: "00000000-0000-0000-0000-000000000001", name: "Ops" };

  it("should migrate a slice persisted under the legacy workspace key", () => {
    expect(
      migrateLegacySlice({ workspace: { ...ZERO_SLICE_STATE, active } }).active,
    ).toEqual(active);
  });

  it("should migrate a slice persisted under the project key", () => {
    expect(
      migrateLegacySlice({ project: { ...ZERO_SLICE_STATE, active } }).active,
    ).toEqual(active);
  });

  it("should default to the zero slice when neither key is present", () => {
    expect(migrateLegacySlice({})).toEqual(ZERO_SLICE_STATE);
  });
});
