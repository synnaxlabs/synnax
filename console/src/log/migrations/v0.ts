// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { z } from "zod";

export const stateZ = z.object({
  key: z.string(),
  version: z.literal("0.0.0"),
  channels: channel.keyZ.array(),
  remoteCreated: z.boolean(),
});

export type State = z.input<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: "0.0.0",
  channels: [],
  remoteCreated: false,
};

export const sliceStateZ = z.object({
  version: z.literal("0.0.0"),
  logs: z.record(stateZ),
});

export type SliceState = z.input<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  logs: {},
};
