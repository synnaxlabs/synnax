// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/cluster/migrations/v1";

export const localCommand = z.enum(["start", "stop"]);
export const localStatus = z.enum(["starting", "running", "stopping", "stopped"]);

const localStateZ = z.object({
  pid: z.number(),
  command: localCommand,
  status: localStatus,
});

export type LocalState = z.input<typeof localStateZ>;

export const ZERO_LOCAL_STATE: LocalState = {
  pid: 0,
  command: "stop",
  status: "stopped",
};

export const sliceStateZ = v1.sliceStateZ.omit({ version: true }).extend({
  version: z.literal("2.0.0"),
  localState: localStateZ,
});

export type SliceState = z.input<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: "2.0.0",
  localState: ZERO_LOCAL_STATE,
  clusters: {
    [v1.LOCAL_CLUSTER_KEY]: v1.LOCAL,
  },
};

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: "cluster.slice",
  migrate: (slice) => ({
    ...slice,
    version: "2.0.0",
    localState: ZERO_LOCAL_STATE,
  }),
});
