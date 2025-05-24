// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/slate/types/v0";

export type NodeProps = v0.NodeProps;
export type State = v0.State;
export type SliceState = v0.SliceState;
export type ToolbarTab = v0.ToolbarTab;
export type ToolbarState = v0.ToolbarState;
export type CopyBuffer = v0.CopyBuffer;
export type AnyState = v0.State;
export type AnySliceState = v0.SliceState;

export const ZERO_STATE = v0.ZERO_STATE;
export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;

const STATE_MIGRATIONS: migrate.Migrations = {};

const SLICE_MIGRATIONS: migrate.Migrations = {};

export const migrateState = migrate.migrator<AnyState, State>({
  name: "slate.state",
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "slate.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z.union([v0.stateZ]).transform((state) => migrateState(state));
