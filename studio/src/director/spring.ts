// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Spring-mass-damper simulation, the animation primitive behind both the synthetic
 * cursor and the virtual camera. Parameters use the web-animation convention
 * (react-spring / Screen Studio): stiffness k, damping c, mass m, with
 * omega0 = sqrt(k/m) and zeta = c / (2 * sqrt(k * m)).
 */
export interface SpringParams {
  stiffness: number;
  damping: number;
  mass: number;
}

export interface SpringState {
  position: number;
  velocity: number;
}

/**
 * step advances the spring one dt toward target using semi-implicit Euler,
 * subdivided for stability at high stiffness.
 */
export const step = (
  state: SpringState,
  target: number,
  params: SpringParams,
  dt: number,
): SpringState => {
  const { stiffness, damping, mass } = params;
  const subSteps = Math.max(1, Math.ceil(dt / 0.004));
  const h = dt / subSteps;
  let { position, velocity } = state;
  for (let i = 0; i < subSteps; i++) {
    const accel = (-stiffness * (position - target) - damping * velocity) / mass;
    velocity += accel * h;
    position += velocity * h;
  }
  return { position, velocity };
};

/**
 * settleLag is the steady-state distance (in seconds) by which a spring trails a
 * constantly moving target; used for phase-lead compensation of the cursor.
 */
export const settleLag = (params: SpringParams): number =>
  params.damping / params.stiffness;

export interface Spring2DState {
  x: SpringState;
  y: SpringState;
}

export const step2D = (
  state: Spring2DState,
  target: { x: number; y: number },
  params: SpringParams,
  dt: number,
): Spring2DState => ({
  x: step(state.x, target.x, params, dt),
  y: step(state.y, target.y, params, dt),
});

export const at = (position: number): SpringState => ({ position, velocity: 0 });

export const at2D = (x: number, y: number): Spring2DState => ({
  x: at(x),
  y: at(y),
});
