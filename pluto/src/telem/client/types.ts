// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel,type Series, TimeRange } from "@synnaxlabs/client";
import { type SeriesDigest } from "@synnaxlabs/x/telem";

export class ReadResponse {
  channel: channel.Payload;
  data: Series[];

  constructor(channel: channel.Payload, data: Series[]) {
    this.channel = channel;
    this.data = data;
  }

  get timeRange(): TimeRange {
    if (this.data.length === 0) return TimeRange.ZERO;
    const first = this.data[0].timeRange;
    const last = this.data[this.data.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }

  get digest(): ReadResponseDigest {
    return {
      channel: this.channel.key,
      timeRange: this.timeRange.toPrettyString(),
      series: this.data.map((s) => s.digest),
    };
  }
}

export interface PromiseFns<T> {
  resolve: (value: T) => void;
  reject: (reason?: any) => void;
}

export interface ReadResponseDigest {
  channel: channel.Key;
  timeRange: string;
  series: SeriesDigest[];
}

export const responseDigests = (responses: ReadResponse[]): ReadResponseDigest[] =>
  responses.map((r) => r.digest);
