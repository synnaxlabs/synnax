// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { desugar, sugar } from "@/sugar";

describe("sugar", () => {
  describe("sugar", () => {
    it("should sugar the provided action correctly", () => {
      const a = {
        type: "myReducer/myAction",
        payload: {},
      };
      const v = sugar(a, "1");
      expect(v).toEqual({
        type: "DA@1://myReducer/myAction",
        payload: {},
      });
    });
  });
  describe("desugar", () => {
    it("should desugar a sugared action into its components", () => {
      const a = {
        type: "DA@1://myReducer/myAction",
        payload: {},
      };
      const { emitted, emitter, action } = desugar(a);
      expect(emitted).toBeTruthy();
      expect(emitter).toBe("1");
      expect(action).toEqual({
        type: "myReducer/myAction",
        payload: {},
      });
    });
    it("should not desugar an action that is not sugared", () => {
      const a = {
        type: "myReducer/myAction",
        payload: {},
      };
      const { emitted, emitter, action } = desugar(a);
      expect(emitted).toBeFalsy();
      expect(emitter).toBeFalsy();
      expect(action).toEqual(a);
    });
  });
});
