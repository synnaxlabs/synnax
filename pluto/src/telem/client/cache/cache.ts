// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Channel, TimeRange } from "@synnaxlabs/client";
import { type Series } from "@synnaxlabs/x";

import { Dynamic } from "@/telem/client/cache/dynamic";
import {
  type DirtyReadResult,
  Static,
  type DirtyReadForWriteResult,
} from "@/telem/client/cache/static";

export class Cache {
  channel: Channel;
  static: Static;
  dynamic: Dynamic;

  constructor(dynamicCap: number, channel: Channel) {
    this.static = new Static();
    this.dynamic = new Dynamic(dynamicCap, channel.dataType);
    this.channel = channel;
  }

  writeDynamic(series: Series[]): Series[] {
    const { flushed, allocated } = this.dynamic.write(series);
    // Buffers that have been flushed out of the dynamic cache are written to the
    // static cache.
    if (flushed.length > 0) {
      const tr = new TimeRange(
        flushed[0].timeRange.start,
        flushed[flushed.length - 1].timeRange.end,
      );
      this.static.write(tr, flushed);
    }
    return allocated;
  }

  writeStatic(tr: TimeRange, series: Series[]): void {
    this.static.write(tr, series);
  }

  dirtyRead(tr: TimeRange): DirtyReadResult {
    return this.static.dirtyRead(tr);
  }

  async dirtyReadForStaticWrite(tr: TimeRange): Promise<DirtyReadForWriteResult> {
    return await this.static.dirtyReadForWrite(tr);
  }
}
