import { describe, expect, it } from "vitest";

import { migrateSlice, ZERO_SLICE_STATE } from "@/layout/migrations";
import * as v0 from "@/layout/migrations/v0";
import * as v3 from "@/layout/migrations/v3";

describe("migrations", () => {
  describe("slice", () => {
    const STATES = [
      v0.ZERO_SLICE_STATE,
      { ...v0.ZERO_SLICE_STATE, version: "0.1.0" },
      { ...v0.ZERO_SLICE_STATE, version: "0.2.0" },
      v3.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
});
