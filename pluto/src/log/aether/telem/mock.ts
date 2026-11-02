// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type status } from "@synnaxlabs/client";
import { type destructor, observe } from "@synnaxlabs/x";

import { telem } from "@/telem/aether";
import { registerInstance, TEST_SOURCE_TYPE } from "@/telem/aether/testutil/factory";

import { type LogEntry, type LogSource, type LogSourceSpec } from "./types";

export class MockLogSource implements LogSource {
  static readonly TYPE = TEST_SOURCE_TYPE;
  status: status.Status = telem.SUCCESS_STATUS;
  private _entries: LogEntry[] = [];
  private activeChannels: Set<channel.Key> = new Set();
  private readonly observable = new observe.BaseObserver<void>();

  value(): telem.Result<LogEntry[]> {
    const { activeChannels, status } = this;
    if (activeChannels.size === 0) return { data: this._entries, status };
    const data = this._entries.filter((e) => activeChannels.has(e.channelKey));
    return { data, status };
  }

  setChannels(channels: channel.Key[]): void {
    this.activeChannels = new Set(channels);
  }

  onChange(handler: observe.Handler<void>): destructor.Destructor {
    return this.observable.onChange(handler);
  }

  push(...entries: LogEntry[]): void {
    this._entries.push(...entries);
  }

  setEntries(entries: LogEntry[]): void {
    this._entries = entries;
  }

  notify(): void {
    this.observable.notify();
  }

  cleanup(): void {
    this._entries = [];
    this.activeChannels.clear();
  }
}

export const registerMockLogSource = (
  testId: string,
  source: MockLogSource,
): destructor.Destructor => registerInstance(testId, source);

export const mockLogSourceSpec = (testId: string): LogSourceSpec => ({
  type: TEST_SOURCE_TYPE,
  props: { testId },
  variant: "source",
  valueType: "log",
});
