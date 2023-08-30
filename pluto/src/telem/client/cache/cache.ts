// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, TimeRange } from "@synnaxlabs/client";
import { Series } from "@synnaxlabs/x";

import { Dynamic } from "@/telem/client/cache/dynamic";
import { Static } from "@/telem/client/cache/static";

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
    const flushed = this.dynamic.write(series);
    if (flushed.length > 0) {
      this.static.write(
        new TimeRange(
          flushed[0].timeRange.start,
          flushed[flushed.length - 1].timeRange.end
        ),
        flushed
      );
      flushed.push(this.dynamic.buffer);
    }
    return flushed;
  }

  writeStatic(tr: TimeRange, series: Series[]): void {
    this.static.write(tr, series);
  }

  read(tr: TimeRange): [Series[], TimeRange[]] {
    const dynamic = this.dynamic.dirtyRead(tr);
    const [staticRes, gaps] = this.static.dirtyRead(tr);
    return [dynamic != null ? staticRes.concat(dynamic) : staticRes, gaps];
  }
}
