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

export const embeddedCommand = z.enum(["start", "stop", "kill"]);
export const embeddedStatus = z.enum([
  "starting",
  "running",
  "stopping",
  "stopped",
  "killed",
]);

const embeddedStateZ = z.object({
  pid: z.number(),
  command: embeddedCommand,
  status: embeddedStatus,
});

export type EmbeddedState = z.input<typeof embeddedStateZ>;

export const ZERO_EMBEDDED_STATE: EmbeddedState = {
  pid: 0,
  command: "stop",
  status: "stopped",
};

export const sliceStateZ = v1.sliceStateZ.omit({ version: true }).extend({
  version: z.literal("2.0.0"),
  localState: embeddedStateZ,
});

export type SliceState = z.input<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: "2.0.0",
  localState: ZERO_EMBEDDED_STATE,
  clusters: {
    [v1.LOCAL_CLUSTER_KEY]: v1.LOCAL,
  },
};

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: "cluster.slice",
  migrate: (slice) => ({
    ...slice,
    version: "2.0.0",
    localState: ZERO_EMBEDDED_STATE,
  }),
});
