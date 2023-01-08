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

import { TelemArray, FrameCache, TimeRange, DataType } from "@synnaxlabs/client";

import { WebGLBufferCache } from "./glCache";
import { FrameRetriever } from "./retriever";

import { Range } from "@/features/workspace";

export class TelemetryClient {
  private readonly glCache: WebGLBufferCache;
  private readonly frameRetriever: FrameRetriever;
  private readonly frameCache: FrameCache;

  constructor(
    glCache: WebGLBufferCache,
    frameRetriever: FrameRetriever,
    frameCache: FrameCache
  ) {
    this.frameCache = frameCache;
    this.glCache = glCache;
    this.frameRetriever = frameRetriever;
  }

  async getFrame(req: TelemetryClientRequest): Promise<TelemetryClientEntry[]> {
    const tr = new TimeRange(req.range.start, req.range.end);
    let { frame, missing } = this.frameCache.get(
      new TimeRange(req.range.start, req.range.end),
      ...req.keys
    );
    if (missing.length > 0)
      frame = await this.frameRetriever.get({
        range: req.range,
        keys: req.keys,
      });

    frame = frame.map((_, a) => {
      let offset: bigint | number = 0;
      if (a.dataType.equals(DataType.TIMESTAMP))
        offset = BigInt(-a.timeRange.start.valueOf());
      a.enrich();
      return a.convert(DataType.FLOAT32, offset);
    });

    this.frameCache.overrideF(tr, frame);
    const entries: TelemetryClientEntry[] = [];

    frame?.entries.forEach(([key, arrays]) => {
      let glBuffers = this.glCache.get(req.range.key, key);
      if (glBuffers == null) glBuffers = this.glCache.set(req.range.key, key, arrays);
      entries.push({
        range: req.range,
        key,
        glBuffers,
        arrays,
      });
    });

    return entries;
  }
}

export interface TelemetryClientRequest {
  range: Range;
  keys: string[];
}

export interface TelemetryClientEntry {
  range: Range;
  key: string;
  glBuffers: WebGLBuffer[];
  arrays: TelemArray[];
}
