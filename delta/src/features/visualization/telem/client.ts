import { TypedArray } from "@synnaxlabs/client";

import { FrameCache } from "./frameCache";
import { WebGLBufferCache } from "./glCache";
import { FrameRetriever } from "./retriever";

import { Range } from "@/features/workspace";

export class TelemetryClient {
  private readonly glCache: WebGLBufferCache;
  private readonly frameCache: FrameCache;
  private readonly frameRetriever: FrameRetriever;

  constructor(
    glCache: WebGLBufferCache,
    frameCache: FrameCache,
    frameRetriever: FrameRetriever
  ) {
    this.glCache = glCache;
    this.frameCache = frameCache;
    this.frameRetriever = frameRetriever;
  }

  async getFrame(req: TelemetryClientRequest): Promise<TelemetryClientEntry[]> {
    // let { frame, missing } = this.frameCache.get({
    //   range: req.range.key,
    //   keys: req.keys,
    // });
    // if (missing.length > 0) {
    const frame = await this.frameRetriever.get({
      range: req.range,
      keys: req.keys,
    });
    // this.frameCache.set({ range: req.range.key, frame: retrieved });
    const entries: TelemetryClientEntry[] = [];
    frame?.forEach(([key, value]) => {
      let glBuffers = this.glCache.get(req.range.key, key);
      if (glBuffers == null) glBuffers = this.glCache.set(req.range.key, key, value);
      entries.push({
        range: req.range,
        key,
        glBuffers,
        arrays: value,
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
  arrays: TypedArray[];
}
