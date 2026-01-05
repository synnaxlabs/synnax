// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, device } from "@synnaxlabs/client";
import { z } from "zod";

const IDENTIFIER_MESSAGE = "Identifier must be between 2-12 characters";

export const nameZ = z.string().min(1, "Name must be at least 1 character long");

export const keyZ = device.keyZ.min(1, "Must specify a device");

export const identifierZ = z
  .string()
  .min(2, IDENTIFIER_MESSAGE)
  .max(12, IDENTIFIER_MESSAGE)
  .regex(
    /^[a-zA-Z][a-zA-Z0-9_]*$/,
    "Identifier must start with a letter and can only contain letters, numbers, and underscores (no spaces or dashes)",
  );

export type Identifier = z.infer<typeof identifierZ>;

export interface CommandStatePair {
  command: channel.Key;
  state: channel.Key;
}
export const ZERO_COMMAND_STATE_PAIR: CommandStatePair = { command: 0, state: 0 };
