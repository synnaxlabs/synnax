// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { synnaxParamsZ } from "@synnaxlabs/client";
import { deep, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/cluster/types/v1";

export const VERSION = "2.0.0";
export type Version = typeof VERSION;

export const clusterZ = synnaxParamsZ
  .extend({ name: z.string().min(1, { message: "Name is required" }) })
  .omit({ connectivityPollFrequency: true, retry: true });
export type Cluster = z.infer<typeof clusterZ> & { key: string };

export type SliceState = Omit<v1.SliceState, "version" | "clusters"> & {
  version: Version;
  clusters: Record<string, Cluster>;
};

type LocalKey = typeof v1.LOCAL_KEY;
const LOCAL: Cluster = {
  ...v1.LOCAL.props,
  key: v1.LOCAL_KEY,
  name: v1.LOCAL.name,
  secure: v1.LOCAL.props.secure ?? false,
};

const DEMO_KEY = "DEMO";
type DemoKey = typeof DEMO_KEY;

const DEMO_CLUSTER = {
  key: DEMO_KEY,
  name: "Demo",
  host: "demo.synnaxlabs.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
} as const;

export type PredefinedClusterKey = LocalKey | DemoKey;

const ZERO_CLUSTERS: Record<string, Cluster> = {
  [v1.LOCAL_KEY]: LOCAL,
  [DEMO_KEY]: DEMO_CLUSTER,
};

export type CoreCluster = Pick<Cluster, "host" | "port">;

export const getPredefinedClusterKey = (
  cluster: CoreCluster,
): PredefinedClusterKey | null => {
  for (const [key, c] of Object.entries(ZERO_CLUSTERS) as [
    PredefinedClusterKey,
    Cluster,
  ][])
    if (cluster.host === c.host && cluster.port == c.port) return key;
  return null;
};

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: VERSION,
  clusters: deep.copy(ZERO_CLUSTERS),
};

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: (slice) => {
    const clusters: Record<string, Cluster> = { [DEMO_KEY]: { ...DEMO_CLUSTER } };
    for (const [key, cluster] of Object.entries(slice.clusters))
      clusters[key] = {
        ...cluster.props,
        name: cluster.name,
        secure: cluster.props.secure ?? false,
        key,
      };

    // check if the LOCAL or DEMO keys are duplicated by a cluster with a real cluster
    // key. If so, delete those keys. This needs to be done after all clusters have been
    // assigned in case the previous slice had a LOCAL key and a cluster with a real key
    // at localhost:9090.
    for (const [key, cluster] of Object.entries(clusters)) {
      const predefinedKey = getPredefinedClusterKey(cluster);
      if (predefinedKey != null && key !== predefinedKey)
        delete clusters[predefinedKey];
    }
    return { ...slice, version: VERSION, clusters };
  },
});
