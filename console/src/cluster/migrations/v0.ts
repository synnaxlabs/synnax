// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { synnaxPropsZ } from "@synnaxlabs/client";
import { z } from "zod";

export const clusterZ = z.object({
  key: z.string(),
  name: z.string(),
  props: synnaxPropsZ,
});

export type Cluster = z.input<typeof clusterZ>;

export const sliceStateZ = z.object({
  version: z.literal("0.0.1"),
  activeCluster: z.string().nullable(),
  clusters: z.record(z.string(), clusterZ),
});

export type SliceState = z.input<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.1",
  activeCluster: null,
  clusters: {},
};
