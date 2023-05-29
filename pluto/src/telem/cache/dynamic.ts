import { DataType, LazyArray, TimeRange, TimeStamp } from "@synnaxlabs/x";

import { GLBufferController } from "@/telem/cache/bufferController";

export class DynamicCache {
  private readonly dataType: DataType;
  curr: LazyArray;
  private readonly cap: number;
  private readonly gl: GLBufferController;

  constructor(gl: GLBufferController, cap: number, dataType: DataType) {
    this.gl = gl;
    this.dataType = dataType;
    this.curr = this.allocate(cap);
    this.cap = cap;
  }

  get length(): number {
    return this.curr.length;
  }

  get data(): LazyArray {
    return this.curr;
  }

  write(arrays: LazyArray[]): LazyArray[] {
    return arrays.flatMap((arr) => this._write(arr));
  }

  read(tr: TimeRange): LazyArray | null {
    if (this.curr.timeRange.overlapsWith(tr)) return this.curr;
    return null;
  }

  private allocate(length: number): LazyArray {
    const tArray = LazyArray.alloc(length, this.dataType, TimeStamp.now().spanRange(0));
    tArray.updateGLBuffer(this.gl);
    return tArray;
  }

  private _write(arr: LazyArray): LazyArray[] {
    const amountWritten = this.curr.write(arr);
    this.curr.updateGLBuffer(this.gl);
    if (amountWritten === arr.length) return [];
    const next = this.allocate(this.cap);
    return [next, ...this._write(arr.slice(amountWritten))];
  }
}
