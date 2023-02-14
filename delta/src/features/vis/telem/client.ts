// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  LazyArray,
  FrameCache,
  TimeRange,
  DataType,
  Synnax,
  Frame,
} from "@synnaxlabs/client";
import { GLDemandCache, GLDemandCacheEntry } from "@synnaxlabs/pluto";

import { Range } from "@/features/workspace";

export class TelemetryClient {
  private readonly glCache: GLDemandCache;
  private readonly client: Synnax;
  private readonly frameCache: FrameCache;

  constructor(glCache: GLDemandCache, client: Synnax, frameCache: FrameCache) {
    this.frameCache = frameCache;
    this.glCache = glCache;
    this.client = client;
  }

  async retrieve(req: TelemetryClientRequest): Promise<TelemetryClientResponse[]> {
    const e: TelemetryClientResponse[] = [];
    for (const r of req.ranges)
      e.push(...(await this.retrieveOne(r, req.keys, req.bypassCache)));
    return e;
  }

  private async retrieveOne(
    range: Range,
    keys: string[],
    bypassCache: boolean = false
  ): Promise<TelemetryClientResponse[]> {
    const tr = new TimeRange(range.start, range.end);
    let frame: Frame = new Frame();
    let missing: string[] = [];
    if (bypassCache) {
      missing = keys;
    } else {
      const res = this.frameCache.get({ tr, keys });
      frame = res.frame;
      missing = res.missing;
    }
    if (missing.length > 0) {
      const remote = this.enrichAndConvertF(await this.readRemote(tr, missing));
      this.frameCache.set(tr, remote);
      this.updateGLCache(range, remote);
      frame = frame.overrideF(remote);
    }
    return frame.entries.map(([key, arrays]) => {
      const buffers = this.glCache.get(this.glCacheKey(range, key));
      if (buffers == null) throw new Error("GLCache is missing buffers");
      return { range, key, arrays, buffers };
    });
  }

  private async readRemote(tr: TimeRange, keys: string[]): Promise<Frame> {
    return await this.client.data.readFrame(tr, keys);
  }

  private enrichAndConvertF(frame: Frame): Frame {
    return frame.map((_, a) => {
      a.enrich();
      if (a.dataType.equals(DataType.TIMESTAMP)) {
        a.offset = BigInt(-a.timeRange.start.valueOf());
      }
      return a;
    });
  }

  private updateGLCache(range: Range, frame: Frame): void {
    frame.entries.forEach(([key, arrays]) =>
      this.glCache.set(
        this.glCacheKey(range, key),
        arrays.map((a) => {
          let offset: bigint | number = 0;
          if (a.dataType.equals(DataType.TIMESTAMP))
            offset = BigInt(-a.timeRange.start.valueOf());
          return a.convert(DataType.FLOAT32, offset);
        })
      )
    );
  }

  private glCacheKey(range: Range, key: string): string {
    return `${range.key}-${key}`;
  }
}

export interface TelemetryClientRequest {
  bypassCache: boolean;
  ranges: Range[];
  keys: string[];
}

export interface TelemetryClientResponse {
  range: Range;
  key: string;
  buffers: GLDemandCacheEntry;
  arrays: LazyArray[];
}
