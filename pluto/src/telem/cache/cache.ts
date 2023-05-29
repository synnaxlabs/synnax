import { Channel, LazyArray, TimeRange } from "@synnaxlabs/client";

import { GLBufferController } from "@/telem/cache/bufferController";
import { DynamicCache } from "@/telem/cache/dynamic";
import { StaticCache } from "@/telem/cache/static";

export class Cache {
  channel: Channel;
  static: StaticCache;
  dynamic: DynamicCache;
  gl: GLBufferController;

  constructor(gl: GLBufferController, dynamicCap: number, channel: Channel) {
    this.gl = gl;
    this.static = new StaticCache();
    this.dynamic = new DynamicCache(gl, dynamicCap, channel.dataType);
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
    arrs.forEach((arr) => arr.updateGLBuffer(this.gl));
    this.static.write(tr, arrs);
  }

  read(tr: TimeRange): [LazyArray[], TimeRange[]] {
    const dynamic = this.dynamic.read(tr);
    const [staticRes, gaps] = this.static.read(tr);
    return [dynamic != null ? staticRes.concat(dynamic) : staticRes, gaps];
  }
}
