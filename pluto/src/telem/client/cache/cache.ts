// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, DataType, LazyArray, TimeRange } from "@synnaxlabs/client";

import { DynamicCache } from "@/telem/client/cache/dynamic";
import { StaticCache } from "@/telem/client/cache/static";

export class ChannelCache {
  channel: Channel;
  static: StaticCache;
  dynamic: DynamicCache;

  constructor(dynamicCap: number, channel: Channel) {
    this.static = new StaticCache();
    this.dynamic = new DynamicCache(dynamicCap, DataType.FLOAT32);
    this.channel = channel;
  }

  writeDynamic(arrs: LazyArray[]): LazyArray[] {
    const flushed = this.dynamic.write(arrs);
    if (flushed.length > 0)
      this.static.write(
        new TimeRange(
          flushed[0].timeRange.start,
          flushed[flushed.length - 1].timeRange.end
        ),
        flushed
      );
    return [...flushed, this.dynamic.buffer];
  }

  writeStatic(tr: TimeRange, arrs: LazyArray[]): void {
    this.static.write(tr, arrs);
  }

  read(tr: TimeRange): [LazyArray[], TimeRange[]] {
    const dynamic = this.dynamic.dirtyRead(tr);
    const [staticRes, gaps] = this.static.dirtyRead(tr);
    return [dynamic != null ? staticRes.concat(dynamic) : staticRes, gaps];
  }
}
