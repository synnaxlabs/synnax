// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/cluster/types/v0";
import * as v1 from "@/cluster/types/v1";
import * as v2 from "@/cluster/types/v2";
import * as v3 from "@/cluster/types/v3";

export const clusterZ = v2.clusterZ;
export type Cluster = v3.Cluster;
export type SliceState = v3.SliceState;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState;

export const getPredefinedClusterKey = v2.getPredefinedClusterKey;

export const ZERO_SLICE_STATE = v3.ZERO_SLICE_STATE;

const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
  [v2.VERSION]: v3.sliceMigration,
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "cluster.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
