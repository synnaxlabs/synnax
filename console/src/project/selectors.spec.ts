// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  selectActive,
  selectActiveKey,
  selectActiveName,
  selectOptionalActive,
  selectOptionalActiveKey,
  selectOptionalActiveName,
} from "@/project/selectors";
import { SLICE_NAME } from "@/project/slice";
import { ZERO_SLICE_STATE } from "@/project/types";

describe("project selectors", () => {
  const active = { key: "00000000-0000-0000-0000-000000000001", name: "Ops" };
  const withActive = { [SLICE_NAME]: { ...ZERO_SLICE_STATE, active } };
  const withoutActive = { [SLICE_NAME]: ZERO_SLICE_STATE };

  describe("with an active project", () => {
    it("should return the active project, key, and name from the required selectors", () => {
      expect(selectActive(withActive)).toEqual(active);
      expect(selectActiveKey(withActive)).toEqual(active.key);
      expect(selectActiveName(withActive)).toEqual(active.name);
    });

    it("should return the same values from the optional selectors", () => {
      expect(selectOptionalActive(withActive)).toEqual(active);
      expect(selectOptionalActiveKey(withActive)).toEqual(active.key);
      expect(selectOptionalActiveName(withActive)).toEqual(active.name);
    });
  });

  describe("without an active project", () => {
    it("should return null from the optional selectors instead of throwing", () => {
      expect(selectOptionalActive(withoutActive)).toBeNull();
      expect(selectOptionalActiveKey(withoutActive)).toBeNull();
      expect(selectOptionalActiveName(withoutActive)).toBeNull();
    });
  });
});
