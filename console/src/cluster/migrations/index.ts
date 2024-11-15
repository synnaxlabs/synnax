// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import type * as v0 from "@/cluster/migrations/v0";
import * as v1 from "@/cluster/migrations/v1";
import * as v2 from "@/cluster/migrations/v2";

export type Cluster = v0.Cluster;
export type SliceState = v2.SliceState;
export type AnySliceState = v0.SliceState | v1.SliceState | v2.SliceState;
export type EmbeddedState = v2.EmbeddedState;

export const LOCAL = v1.LOCAL;
export const LOCAL_CLUSTER_KEY = v1.LOCAL_CLUSTER_KEY;
export const LOCAL_PROPS = v1.LOCAL_PROPS;
export const isLocalCluster = v1.isLocalCluster;

export const ZERO_SLICE_STATE = v2.ZERO_SLICE_STATE;

export const SLICE_MIGRATIONS: migrate.Migrations = {
  "0.0.1": v1.sliceMigration,
  "1.0.0": v2.sliceMigration,
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "cluster.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
