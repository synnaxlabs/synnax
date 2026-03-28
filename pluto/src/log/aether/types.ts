// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type observe } from "@synnaxlabs/x";
import { z } from "zod";

import { type Source, sourceSpecZ, type Telem } from "@/telem/aether/telem";

export interface LogEntry {
  channelKey: number;
  timestamp: bigint;
  value: string;
}

export interface LogSource extends Source<LogEntry[]> {
  readonly evictedCount: number;
  setChannels?: (channels: Array<number | string>) => void;
}

export const logSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("log") });
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;

class NoopLogSource implements LogSource {
  static readonly TYPE = "noop-log-source";
  readonly evictedCount = 0;

  value(): LogEntry[] {
    return [];
  }

  cleanup(): void {}

  onChange(_handler: observe.Handler<void>): destructor.Destructor {
    return () => {};
  }
}

export const noopLogSourceSpec: LogSourceSpec = {
  type: NoopLogSource.TYPE,
  props: {},
  variant: "source",
  valueType: "log",
};

export const NOOP_LOG_REGISTRY: Record<string, new () => Telem> = {
  [NoopLogSource.TYPE]: NoopLogSource as unknown as new () => Telem,
};
