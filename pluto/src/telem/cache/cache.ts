// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, LazyArray, TimeRange } from "@synnaxlabs/client";

import { DynamicCache } from "@/telem/cache/dynamic";
import { StaticCache } from "@/telem/cache/static";

export class Cache {
  channel: Channel;
  static: StaticCache;
  dynamic: DynamicCache;

  constructor(dynamicCap: number, channel: Channel) {
    this.static = new StaticCache();
    this.dynamic = new DynamicCache(dynamicCap, channel.dataType);
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
    return [...flushed, this.dynamic.curr];
  }

  writeStatic(tr: TimeRange, arrs: LazyArray[]): void {
    this.static.write(tr, arrs);
  }

  read(tr: TimeRange): [LazyArray[], TimeRange[]] {
    const dynamic = this.dynamic.read(tr);
    const [staticRes, gaps] = this.static.read(tr);
    return [dynamic != null ? staticRes.concat(dynamic) : staticRes, gaps];
  }
}
