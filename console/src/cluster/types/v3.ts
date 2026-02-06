// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, uuid } from "@synnaxlabs/x";

import * as v2 from "@/cluster/types/v2";

export const VERSION = "3.0.0";
export type Version = typeof VERSION;

export type Cluster = v2.Cluster;
export type SliceState = Omit<v2.SliceState, "version"> & { version: Version };

export const ZERO_SLICE_STATE: SliceState = {
  ...v2.ZERO_SLICE_STATE,
  version: VERSION,
};

export const sliceMigration = migrate.createMigration<v2.SliceState, SliceState>({
  name: "cluster.slice",
  migrate: (slice) => {
    const clusters: Record<string, Cluster> = {};
    for (const [key, cluster] of Object.entries(slice.clusters)) {
      if (key.length === 0) {
        const newKey = uuid.create();
        clusters[newKey] = { ...cluster, key: newKey };
      } else clusters[key] = cluster;
    }
    return { ...slice, version: VERSION, clusters };
  },
});
