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
import { type DirtyReadResult, Static } from "@/telem/client/cache/static";

export class Unary {
  readonly channel: channel.Payload;
  private closed: boolean = false;
  private readonly ins: alamos.Instrumentation;
  private readonly static: Static;
  private readonly dynamic: Dynamic;

  constructor(
    dynamicCap: number,
    channel: channel.Payload,
    ins: alamos.Instrumentation,
  ) {
    this.ins = ins;
    this.static = new Static(ins);
    this.dynamic = new Dynamic(dynamicCap, channel.dataType);
    this.channel = channel;
  }

  writeDynamic(series: Series[]): Series[] {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dynamic write to a closed cache for channel ${this.channel.name}`,
      );
      return [];
    }
    const { flushed, allocated } = this.dynamic.write(series);
    // Buffers that have been flushed out of the dynamic cache are written to the
    // static cache.
    if (flushed.length > 0) this.static.write(flushed);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    return this.dynamic.buffer;
  }

  writeStatic(series: Series[]): void {
    if (this.closed)
      return this.ins.L.warn(
        `Ignoring attempted static write to a closed cache for channel ${this.channel.name}`,
      );
    this.static.write(series);
  }

  read(tr: TimeRange): DirtyReadResult {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dirty read from a closed cache for channel ${this.channel.name}`,
      );
      return { series: [], gaps: [] };
    }
    return this.static.dirtyRead(tr);
  }

  garbageCollect(): void {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted garbage collection on a closed cache for channel ${this.channel.name}`,
      );
    }
    this.static.garbageCollect();
  }

  close(): void {
    this.closed = true;
    this.dynamic.close();
    this.static.close();
  }
}
