// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, rack, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const TYPE = "sequence";
export const typeZ = z.literal(TYPE);
export type Type = z.infer<typeof typeZ>;

export const configZ = z.object({
  rate: z.number().min(1),
  rack: rack.keyZ.min(1, "Must select a location to connect from"),
  read: z.array(channel.keyZ),
  write: z.array(channel.keyZ),
  script: z.string(),
  globals: z.record(z.string(), z.unknown()),
});
export type Config = z.infer<typeof configZ>;
export const ZERO_CONFIG: Config = {
  rate: 10,
  rack: 0,
  read: [],
  write: [],
  script: `-- Edit your control sequence here.
-- To access a channel value, simply type in its name or use the get('channel-name') function.
-- To set a channel value, use the set('channel-name', value) function.
-- For further documentation, see https://docs.synnaxlabs.com/reference/control/embedded/get-started`,
  globals: {},
};

export const statusDetailsZ = z.unknown();

export type Task = task.Task<typeof typeZ, typeof configZ, typeof statusDetailsZ>;
export type Payload = task.Payload<typeof typeZ, typeof configZ, typeof statusDetailsZ>;

export const ZERO_PAYLOAD: Payload = {
  key: "",
  name: "Control Sequence",
  config: ZERO_CONFIG,
  type: TYPE,
};
