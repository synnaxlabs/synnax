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

import { type Source, sourceSpecZ } from "@/telem/aether/telem";

export interface LogEntry {
  channelKey: channel.Key;
  timestamp: TimeStamp;
  value: string;
}

export interface LogSource extends Source<LogEntry[]> {
  // Required on the interface (not just StreamMultiChannelLog) because the aether
  // Log class uses it to adjust scroll offset and selection after GC.
  readonly evictedCount: number;
  setChannels?: (channels: channel.Key[]) => void;
}

export const logSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("log") });
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;
