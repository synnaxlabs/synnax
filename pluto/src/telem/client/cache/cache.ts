// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type alamos } from "@synnaxlabs/alamos";
import { type channel, type TimeRange } from "@synnaxlabs/client";
import { type Series } from "@synnaxlabs/x";

import { Dynamic } from "@/telem/client/cache/dynamic";
import {
  type DirtyReadResult,
  Static,
  type DirtyReadForWriteResult,
} from "@/telem/client/cache/static";

export class Cache {
  readonly channel: channel.Payload;
  readonly static: Static;
  readonly dynamic: Dynamic;

  constructor(
    dynamicCap: number,
    channel: channel.Payload,
    ins: alamos.Instrumentation,
  ) {
    this.static = new Static(ins);
    this.dynamic = new Dynamic(dynamicCap, channel.dataType);
    this.channel = channel;
  }

  writeDynamic(series: Series[]): Series[] {
    const { flushed, allocated } = this.dynamic.write(series);
    // Buffers that have been flushed out of the dynamic cache are written to the
    // static cache.
    if (flushed.length > 0) this.static.write(flushed);
    return allocated;
  }

  writeStatic(series: Series[]): void {
    this.static.write(series);
  }

  dirtyRead(tr: TimeRange): DirtyReadResult {
    return this.static.dirtyRead(tr);
  }

  async dirtyReadForStaticWrite(tr: TimeRange): Promise<DirtyReadForWriteResult> {
    return await this.static.dirtyReadForWrite(tr);
  }

  close(): void {
    this.dynamic.close();
    this.static.close();
  }
}
