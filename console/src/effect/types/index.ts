// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/effect/types/v0";

export type Effect = v0.Effect;
export type SliceState = v0.SliceState;

export type AnyEffect = v0.Effect;
export type AnySliceState = v0.SliceState;

export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;
export const ZERO_EFFECT = v0.ZERO_EFFECT;

export const EFFECT_MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "effect",
  migrations: EFFECT_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
