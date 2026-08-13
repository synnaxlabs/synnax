// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Cluster, SLICE_NAME } from "@/session/cluster/slice";
import { type ConsolePreloadedState } from "@/testutil";

/** Connection parameters for the local test core every live-core spec runs against. */
export const CONNECTION_PARAMS: Omit<Cluster, "key" | "name"> = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

export const createCluster = (
  key: string,
  overrides: Partial<Cluster> = {},
): Cluster => ({ key, name: key, ...CONNECTION_PARAMS, ...overrides });

export const createClusterState = (
  clusters: Cluster[],
  selected?: string,
): ConsolePreloadedState => ({
  [SLICE_NAME]: {
    version: 0,
    selected,
    clusters: Object.fromEntries(clusters.map((c) => [c.key, c])),
  },
});
