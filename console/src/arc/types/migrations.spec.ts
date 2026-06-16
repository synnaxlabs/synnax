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
  type AnySliceState,
  type AnyState,
  anyStateZ,
  migrateSlice,
  migrateState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/arc/types";
import * as v0 from "@/arc/types/v0";
import * as v1 from "@/arc/types/v1";
import * as v2 from "@/arc/types/v2";

const STATES: AnyState[] = [v0.ZERO_STATE, v1.ZERO_STATE, v2.ZERO_STATE];
const SLICE_STATES: AnySliceState[] = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
];

describe("arc type migrations", () => {
  describe("state", () => {
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated.version).toBe(v2.VERSION);
      });
    });
  });

  describe("slice", () => {
    SLICE_STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });

  describe("v1 → v2 set_status rewrite", () => {
    const node = (key: string, extras: Record<string, unknown> = {}): v1.NodeProps => ({
      ...extras,
      key,
    });
    const legacySetStatus = (extras: Record<string, unknown> = {}): v1.NodeProps =>
      node("set_status", extras);
    const buildV1StateWithProps = (props: Record<string, v1.NodeProps>): v1.State => ({
      ...v1.ZERO_STATE,
      key: "test",
      graph: { ...v1.ZERO_GRAPH_STATE, props },
    });

    it("should rewrite a set_status node with all legacy fields", () => {
      const v1State = buildV1StateWithProps({
        n1: legacySetStatus({
          statusKey: "alarm",
          variant: "warning",
          message: "overpressure",
        }),
      });
      const migrated = migrateState(v1State);
      expect(migrated.graph.props.n1).toEqual({
        key: "status.set",
        key_or_name: "alarm",
        variant: "warning",
        message: "overpressure",
      });
    });

    it("should default missing legacy fields when rewriting set_status", () => {
      const v1State = buildV1StateWithProps({ n1: legacySetStatus() });
      const migrated = migrateState(v1State);
      expect(migrated.graph.props.n1).toEqual({
        key: "status.set",
        key_or_name: "",
        variant: "success",
        message: "",
      });
    });

    it("should pass through non-set_status nodes unchanged", () => {
      const v1State = buildV1StateWithProps({
        n1: node("status.set", {
          key_or_name: "x",
          variant: "info",
          message: "m",
        }),
        n2: node("channel.read", { channel: 42 }),
      });
      const migrated = migrateState(v1State);
      expect(migrated.graph.props.n1).toEqual(v1State.graph.props.n1);
      expect(migrated.graph.props.n2).toEqual(v1State.graph.props.n2);
    });

    it("should rewrite set_status nodes across multiple arcs in a slice", () => {
      const v1Slice: v1.SliceState = {
        ...v1.ZERO_SLICE_STATE,
        arcs: {
          a: buildV1StateWithProps({
            n: legacySetStatus({ statusKey: "a", variant: "error", message: "boom" }),
          }),
          b: buildV1StateWithProps({ n: legacySetStatus({ statusKey: "b" }) }),
        },
      };
      const migrated = migrateSlice(v1Slice);
      expect(migrated.arcs.a.graph.props.n).toMatchObject({
        key: "status.set",
        key_or_name: "a",
        variant: "error",
        message: "boom",
      });
      expect(migrated.arcs.b.graph.props.n).toMatchObject({
        key: "status.set",
        key_or_name: "b",
        variant: "success",
        message: "",
      });
    });
  });

  describe("anyStateZ", () => {
    it("should parse and migrate v0 state to v2", () => {
      const result = anyStateZ.parse(v0.ZERO_STATE);
      expect(result.version).toBe(v2.VERSION);
    });

    it("should parse and migrate v1 state to v2", () => {
      const result = anyStateZ.parse(v1.ZERO_STATE);
      expect(result.version).toBe(v2.VERSION);
    });

    it("should parse v2 state as-is", () => {
      const result = anyStateZ.parse(ZERO_STATE);
      expect(result.version).toBe(v2.VERSION);
    });
  });
});
