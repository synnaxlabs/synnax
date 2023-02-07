/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import {
  TArray,
  FrameCache,
  TimeRange,
  DataType,
  Synnax,
  Frame,
} from "@synnaxlabs/client";

import { GLBufferCache } from "./glCache";

import { Range } from "@/features/workspace";
import { w } from "@tauri-apps/api/event-2a9960e7";

export class TelemetryClient {
  private readonly glCache: GLBufferCache;
  private readonly client: Synnax;
  private readonly frameCache: FrameCache;

  constructor(glCache: GLBufferCache, client: Synnax, frameCache: FrameCache) {
    this.frameCache = frameCache;
    this.glCache = glCache;
    this.client = client;
  }

  async retrieve(req: TelemetryClientRequest): Promise<TelemetryClientResponse[]> {
    const e: TelemetryClientResponse[] = [];
    for (const r of req.ranges) e.push(...(await this.retrieveOne(r, req.keys)));
    return e;
  }

  private async retrieveOne(
    range: Range,
    keys: string[]
  ): Promise<TelemetryClientResponse[]> {
    const tr = new TimeRange(range.start, range.end);
    let { frame, missing } = this.frameCache.get(tr, ...keys);
    if (missing.length > 0) frame = frame.overrideF(await this.readRemote(tr, missing));
    return frame.entries.map(([key, arrays]) => ({
      range,
      key,
      arrays,
      ...this.getAndUpdateGLCache(range, key, arrays)
    }));
  }

  private async readRemote(tr: TimeRange, keys: string[]): Promise<Frame> {
    const frame = await this.client.data.readFrame(tr, keys);
    this.enrich(frame);
    this.frameCache.overrideF(tr, frame);
    return frame;
  }

  private getAndUpdateGLCache(
    range: Range,
    key: string,
    arrays: TArray[]
  ): {glBuffers: WebGLBuffer[], glOffsets: Array<number | bigint>} {
    let glBuffers = this.glCache.get(range.key, key);
    let glOffsets: Array<number | bigint> = [];
    arrays = arrays.map((a) => {
      let offset: bigint | number = 0;
      if (a.dataType.equals(DataType.TIMESTAMP)) 
        offset = Number(-a.timeRange.start.valueOf())
      glOffsets.push(offset);
      return a.convert(DataType.FLOAT32, offset)
    });
    if (glBuffers == null) glBuffers = this.glCache.set(range.key, key, arrays);
    return  {glBuffers, glOffsets};
  }

  private enrich(f: Frame): void {
    f.arrays.forEach((a) => a.enrich());
  }
}

export interface TelemetryClientRequest {
  ranges: Range[];
  keys: string[];
}

export interface TelemetryClientResponse {
  range: Range;
  key: string;
  glBuffers: WebGLBuffer[];
  glOffsets: Array<number | bigint>;
  arrays: TArray[];
}
