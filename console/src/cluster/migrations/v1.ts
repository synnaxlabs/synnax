// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SynnaxProps } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/cluster/migrations/v0";

export const sliceStateZ = v0.sliceStateZ.omit({ version: true }).extend({
  version: z.literal("1.0.0"),
});

export type SliceState = z.input<typeof sliceStateZ>;

export const LOCAL_PROPS: SynnaxProps = {
  name: "Local",
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

export const isLocalCluster = (props: any): boolean =>
  props.host === LOCAL_PROPS.host &&
  props.port == LOCAL_PROPS.port &&
  props.username === LOCAL_PROPS.username &&
  props.password === LOCAL_PROPS.password &&
  props.secure === LOCAL_PROPS.secure;

export const LOCAL_CLUSTER_KEY = "LOCAL";

export const LOCAL: v0.Cluster = {
  key: LOCAL_CLUSTER_KEY,
  name: "Local",
  props: LOCAL_PROPS,
};

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: "1.0.0",
  clusters: {
    [LOCAL_CLUSTER_KEY]: LOCAL,
  },
};

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: "cluster.slice",
  migrate: (slice) => ({
    ...slice,
    version: "1.0.0",
    clusters: {
      ...slice.clusters,
      [LOCAL_CLUSTER_KEY]: LOCAL,
    },
  }),
});
