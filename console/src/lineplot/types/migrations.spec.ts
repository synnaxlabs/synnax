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
  anyStateZ,
  migrateSlice,
  migrateState,
  stateZ,
  ZERO_ANNOTATIONS_STATE,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/lineplot/types";
import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";
import * as v2 from "@/lineplot/types/v2";
import * as v3 from "@/lineplot/types/v3";
import * as v4 from "@/lineplot/types/v4";
import * as v5 from "@/lineplot/types/v5";

describe("migrations", () => {
  describe("state", () => {
    const STATES = [
      v0.ZERO_STATE,
      v1.ZERO_STATE,
      v2.ZERO_STATE,
      v3.ZERO_STATE,
      v4.ZERO_STATE,
      v5.ZERO_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated.version).toBe(ZERO_STATE.version);
        expect(migrated.selectedRules).toEqual([]);
        expect(migrated.viewport).toBeDefined();
        expect(migrated.toolbar).toBeDefined();
      });
    });

    it("should park v4 body into pendingUpload when remoteCreated is false", () => {
      const migrated = migrateState({ ...v4.ZERO_STATE, remoteCreated: false });
      expect(migrated.pendingUpload).toBeDefined();
      expect(migrated.pendingUpload?.title).toBeDefined();
      expect(migrated.pendingUpload?.legend).toBeDefined();
      expect(migrated.pendingUpload?.axes).toBeDefined();
    });

    it("should leave pendingUpload undefined when remoteCreated is true", () => {
      const migrated = migrateState({ ...v4.ZERO_STATE, remoteCreated: true });
      expect(migrated.pendingUpload).toBeUndefined();
    });

    it("should lift selected rules into selectedRules on v4 to v5 migration", () => {
      const state: v4.State = {
        ...v4.ZERO_STATE,
        remoteCreated: true,
        rules: [
          { ...v0.ZERO_RULE_STATE, key: "r1", selected: true },
          { ...v0.ZERO_RULE_STATE, key: "r2", selected: false },
          { ...v0.ZERO_RULE_STATE, key: "r3", selected: true },
        ],
      };
      const migrated = migrateState(state);
      expect(migrated.selectedRules).toEqual(["r1", "r3"]);
    });
  });

  describe("slice", () => {
    const STATES = [
      v0.ZERO_SLICE_STATE,
      v1.ZERO_SLICE_STATE,
      v2.ZERO_SLICE_STATE,
      v3.ZERO_SLICE_STATE,
      v4.ZERO_SLICE_STATE,
      v5.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated.version).toBe(ZERO_SLICE_STATE.version);
        expect(migrated.plots).toEqual({});
      });
    });
  });

  describe("v4 annotations backward compatibility", () => {
    it("should fill annotations with default when absent from a persisted v4 state", () => {
      const { annotations: _omit, ...persisted } = v4.ZERO_STATE;
      expect(v4.stateZ.parse(persisted).annotations).toEqual(ZERO_ANNOTATIONS_STATE);
    });

    it("should load a persisted v5 state via anyStateZ", () => {
      expect(stateZ.parse(ZERO_STATE)).toEqual(ZERO_STATE);
      expect(anyStateZ.parse(ZERO_STATE)).toEqual(ZERO_STATE);
    });
  });
});
