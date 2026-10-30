// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type TimeStamp } from "@synnaxlabs/client";
import { z } from "zod";

import { telem } from "@/telem/aether";

export interface LogEntry {
  channelKey: channel.Key;
  timestamp: TimeStamp;
  value: string;
  // True when this entry was produced by splitting a sample on \n;
  continuation?: boolean;
}

export interface LogSource extends telem.Source<LogEntry[]> {
  setChannels?: (channels: channel.Key[]) => void;
}

export const logSourceSpecZ = telem.sourceSpecZ.extend({ valueType: z.literal("log") });
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;
