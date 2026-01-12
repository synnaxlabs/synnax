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

export const VERSION = "1.0.0";
export type Version = typeof VERSION;

export type SliceState = Omit<v0.SliceState, "version"> & { version: Version };

export const LOCAL_KEY = "LOCAL";
const LOCAL_NAME = "Local";
export const LOCAL: v0.Cluster = {
  key: LOCAL_KEY,
  name: LOCAL_NAME,
  props: {
    name: LOCAL_NAME,
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
  },
};

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: VERSION,
  clusters: { [LOCAL_KEY]: LOCAL },
};

export const SLICE_MIGRATION_NAME = "cluster.slice";

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: (slice) => ({
    ...slice,
    version: VERSION,
    clusters: { ...slice.clusters, [LOCAL_KEY]: LOCAL },
  }),
});
