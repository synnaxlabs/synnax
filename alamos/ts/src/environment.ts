// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A list of valid environments */
export const ENVIRONMENTS = ["bench", "debug", "prod"] as const;

/**
 * Defines the environment in whcih instrumentation in running. Traces can
 * be constrained to run only in certain environments.
 */
export type Environment = (typeof ENVIRONMENTS)[number];

/**
 * Takes an environment and returns true if it is valid for use i.e.
 * 'should this trace be executed?
 */
export type EnvironmentFilter = (env: Environment) => boolean;

/**
 * @param treshold
 * @returns An environnment filter that returns true if the environment is greater
 * than or equal to the given threshold.
 */
export const envThresholdFilter =
  (treshold: Environment): EnvironmentFilter =>
  (env: Environment) =>
    ENVIRONMENTS.indexOf(env) >= ENVIRONMENTS.indexOf(treshold);
