import { Channel, LazyArray, TimeRange } from "@synnaxlabs/client";

import { DynamicCache } from "@/telem/cache/dynamic";
import { StaticCache } from "@/telem/cache/static";
import { VisArray } from "@/telem/visArray";
import { GLBufferController } from "@/core/vis/telem/bufferController";

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

  writeDynamic(arrs: LazyArray[]): VisArray[] {
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

  writeStatic(tr: TimeRange, arrs: LazyArray[]): VisArray[] {
    const vis = arrs.map((arr) => {
      const buf = this.gl.createBuffer();
      if (buf == null) throw new Error("Failed to create buffer");
      this.gl.bindBuffer(this.gl.ARRAY_BUFFER, buf);
      this.gl.bufferData(this.gl.ARRAY_BUFFER, arr.buffer, this.gl.STATIC_DRAW);
      return new VisArray(buf, arr);
    });
    this.static.write(tr, vis);
    return vis;
  }

  read(tr: TimeRange): [VisArray[], TimeRange[]] {
    const dynamic = this.dynamic.read(tr);
    const [staticRes, gaps] = this.static.read(tr);
    return [dynamic != null ? staticRes.concat(dynamic) : staticRes, gaps];
  }
}
