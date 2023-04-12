// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { useAsyncEffect } from "@synnaxlabs/pluto";
import { Deep, unique } from "@synnaxlabs/x";

import { GOOD_STATUS, Status, StatusProvider } from "./core";

import { AxisKey, Y_AXIS_KEYS } from "@/vis/axis";
import { Channels } from "@/vis/line/channels";
import { Ranges } from "@/vis/line/ranges";
import {
  useTelemetryClient,
  TelemetryClient,
  TelemetryClientResponse,
} from "@/vis/telem";

const ZERO_DATA: InternalState = {
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  x1: [],
  x2: [],
};

type InternalState = Record<AxisKey, TelemetryClientResponse[]>;

export class Data implements StatusProvider {
  private readonly entries: InternalState;
  readonly status: Status;

  constructor(entries: InternalState, status: Status = GOOD_STATUS) {
    this.entries = entries;
    this.status = status;
  }

  static use(channels: Channels, ranges: Ranges): Data {
    const client = useTelemetryClient();
    const [data, setData] = useState<Data>(new Data(ZERO_DATA));

    useAsyncEffect(async () => {
      if (client === null) return;
      if (data.isZero)
        setData(
          new Data(ZERO_DATA, {
            display: true,
            variant: "loading",
            children: "Loading...",
          })
        );
      setData(await Data.fetch(channels, ranges, client));
    }, [client, channels, ranges]);

    return data;
  }

  get isZero(): boolean {
    return Object.values(this.entries).every((e) => e.length === 0);
  }

  static async fetch(
    channels: Channels,
    ranges: Ranges,
    client: TelemetryClient
  ): Promise<Data> {
    let entries: TelemetryClientResponse[] = [];
    let error: Error | null = null;
    try {
      entries = await client.retrieve({
        keys: channels.keys,
        ranges: ranges.array,
        bypassCache: ranges.isLive,
      });
    } catch (err) {
      error = err as Error;
    }
    const core = Deep.copy(ZERO_DATA);
    ranges.forEach((range) =>
      channels.forEachAxis((channels, axis) => {
        const keys = channels.map((c) => c.key);
        core[axis].push(
          ...entries.filter((e) => keys.includes(e.key) && e.range.key === range.key)
        );
      })
    );
    if (error != null) {
      return new Data(core, {
        display: true,
        variant: "error",
        children: error.message,
      });
    }

    const empty = Y_AXIS_KEYS.every((axis) =>
      core[axis].every((res) => res.arrays.every((array) => array.length === 0))
    );

    if (empty)
      return new Data(core, {
        display: true,
        variant: "warning",
        children: "No data found.",
      });

    return new Data(core, GOOD_STATUS);
  }

  axis(key: AxisKey): TelemetryClientResponse[] {
    return this.entries[key];
  }

  forEachAxis(fn: (key: AxisKey, data: TelemetryClientResponse[]) => void): void {
    Object.entries(this.entries).forEach(([key, data]) => fn(key as AxisKey, data));
  }

  forEachChannel(
    fn: (ch: string, axis: AxisKey, data: TelemetryClientResponse[]) => void
  ): void {
    Object.entries(this.entries).forEach(([axis, data]) => {
      const keys = unique(data.map((d) => d.key));
      keys.forEach((key) =>
        fn(
          key,
          axis as AxisKey,
          data.filter((d) => d.key === key)
        )
      );
    });
  }
}
