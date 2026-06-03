// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import {
  anyStateZ,
  fromWire,
  migrateSlice,
  migrateState,
  stateZ,
  toWire,
  ZERO_ANNOTATIONS_STATE,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/lineplot/types";
import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";
import * as v2 from "@/lineplot/types/v2";
import * as v3 from "@/lineplot/types/v3";
import * as v4 from "@/lineplot/types/v4";

describe("migrations", () => {
  describe("state", () => {
    const STATES = [
      v0.ZERO_STATE,
      v1.ZERO_STATE,
      v2.ZERO_STATE,
      v3.ZERO_STATE,
      v4.ZERO_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        expect(migrateState(state)).toEqual(ZERO_STATE);
      });
    });
  });
  describe("slice", () => {
    const STATES = [
      v0.ZERO_SLICE_STATE,
      v1.ZERO_SLICE_STATE,
      v2.ZERO_SLICE_STATE,
      v3.ZERO_SLICE_STATE,
      v4.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        expect(migrateSlice(state)).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
  describe("v4 annotations backward compatibility", () => {
    it("should fill annotations with default when absent from a persisted v4 state", () => {
      const { annotations: _omit, ...persisted } = ZERO_STATE;
      expect(stateZ.parse(persisted).annotations).toEqual(ZERO_ANNOTATIONS_STATE);
    });
    it("should load a persisted v4 state without annotations via anyStateZ", () => {
      const { annotations: _omit, ...persisted } = ZERO_STATE;
      expect(anyStateZ.parse(persisted)).toEqual(ZERO_STATE);
    });
  });
  describe("wire projection", () => {
    it("should drop UI-only fields when projecting a State to the server body", () => {
      const wire = toWire(ZERO_STATE);
      expect(wire).toEqual({
        title: ZERO_STATE.title,
        legend: ZERO_STATE.legend,
        channels: ZERO_STATE.channels,
        ranges: ZERO_STATE.ranges,
        axes: ZERO_STATE.axes.axes,
        lines: ZERO_STATE.lines,
        rules: ZERO_STATE.rules,
      });
      expect(wire).not.toHaveProperty("key");
      expect(wire).not.toHaveProperty("version");
      expect(wire).not.toHaveProperty("annotations");
    });
    it("should hydrate a server body into a remote-created State", () => {
      const lp: lineplot.LinePlot = { ...toWire(ZERO_STATE), key: "plot-1", name: "n" };
      expect(fromWire(lp)).toEqual({
        ...ZERO_STATE,
        key: "plot-1",
        remoteCreated: true,
      });
    });
  });
});
