// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const TYPE = "sequence";
export type Type = typeof TYPE;

export const configZ = z.object({
  rate: z.number().min(1),
  read: z.array(channel.keyZ),
  write: z.array(channel.keyZ),
  script: z.string(),
  globals: z.record(z.string(), z.unknown()),
});
export type Config = z.infer<typeof configZ>;
export const ZERO_CONFIG: Config = {
  rate: 10,
  read: [],
  write: [],
  script: "",
  globals: {},
};

export const stateDetailsZ = z.object({ running: z.boolean(), message: z.string() });
export type StateDetails = z.infer<typeof stateDetailsZ>;

export type Task = task.Task<Config, StateDetails, Type>;
export type Payload = task.Payload<Config, StateDetails, Type>;

export const ZERO_PAYLOAD: Payload = {
  key: "",
  name: "Control Sequence",
  config: ZERO_CONFIG,
  type: TYPE,
};
