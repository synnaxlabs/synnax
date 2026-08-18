// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CAMERA_SPRING, CURSOR_SPRING } from "@/director/constants";
import { at, settleLag, step } from "@/director/spring";

const settle = (
  params: typeof CURSOR_SPRING,
  target: number,
  seconds: number,
): number => {
  let state = at(0);
  const dt = 1 / 60;
  for (let i = 0; i < seconds * 60; i++) state = step(state, target, params, dt);
  return state.position;
};

describe("spring", () => {
  it("should converge to the target", () => {
    expect(settle(CURSOR_SPRING, 100, 2)).toBeCloseTo(100, 1);
    expect(settle(CAMERA_SPRING, 2, 3)).toBeCloseTo(2, 2);
  });

  it("should overshoot slightly with the underdamped camera spring", () => {
    let state = at(0);
    let max = 0;
    const dt = 1 / 60;
    for (let i = 0; i < 240; i++) {
      state = step(state, 1, CAMERA_SPRING, dt);
      max = Math.max(max, state.position);
    }
    expect(max).toBeGreaterThan(1);
    expect(max).toBeLessThan(1.08);
  });

  it("should be stable at high stiffness", () => {
    const stiff = { stiffness: 1000, damping: 40, mass: 1 };
    let state = at(0);
    const dt = 1 / 60;
    for (let i = 0; i < 600; i++) state = step(state, 50, stiff, dt);
    expect(Number.isFinite(state.position)).toBe(true);
    expect(state.position).toBeCloseTo(50, 1);
  });

  it("should report the documented phase lag for the cursor spring", () => {
    expect(settleLag(CURSOR_SPRING)).toBeCloseTo(70 / 470, 5);
  });

  it("should carry velocity across retargets without discontinuity", () => {
    let state = at(0);
    const dt = 1 / 60;
    for (let i = 0; i < 20; i++) state = step(state, 100, CURSOR_SPRING, dt);
    const before = state.position;
    state = step(state, -100, CURSOR_SPRING, dt);
    expect(Math.abs(state.position - before)).toBeLessThan(
      Math.abs(state.velocity) * dt + 1,
    );
  });
});
