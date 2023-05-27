import { DataType, LazyArray, TimeRange, TimeStamp } from "@synnaxlabs/x";

import { VisArray } from "@/telem/visArray";
import { GLBufferController } from "@/core/vis/gl/bufferController";

export class DynamicCache {
  private readonly dataType: DataType;
  curr: VisArray;
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
    return this.curr.arr;
  }

  private allocate(length: number): VisArray {
    const tArray = LazyArray.alloc(length, this.dataType, TimeStamp.now().spanRange(0));
    const glBuffer = this.gl.createBuffer();
    if (glBuffer == null) throw new Error("Failed to create buffer");
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, glBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.dataType.density.size(length).valueOf(),
      this.gl.DYNAMIC_DRAW
    );
    const vArr = new VisArray(glBuffer, tArray);
    vArr.length = 0;
    return vArr;
  }

  private _write(arr: LazyArray, offset: number): VisArray[] {
    const amountWritten =
      this.curr.available < arr.length - offset
        ? this.curr.available
        : arr.length - offset;

    this.curr.arr.set(arr.data.slice(offset, offset + amountWritten), this.curr.length);
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.curr.gl);
    const glOffset = this.dataType.density.size(offset).valueOf();
    const glLength = this.dataType.density.size(amountWritten).valueOf();
    this.gl.bufferSubData(
      this.gl.ARRAY_BUFFER,
      this.byteLength,
      arr.data.buffer,
      glOffset,
      glLength
    );
    this.curr.length += amountWritten;
    this.curr.timeRange.end = TimeStamp.now();

    const leftToWrite = arr.length - offset - amountWritten;
    if (leftToWrite <= 0) return [];

    const toFlush = [this.curr];
    this.curr = this.allocate(this.cap);
    toFlush.push(...this._write(arr, offset + amountWritten));
    return toFlush;
  }

  private get byteLength(): number {
    return this.dataType.density.size(this.curr.length).valueOf();
  }

  write(arrays: LazyArray[]): VisArray[] {
    return arrays.flatMap((arr) => this._write(arr, 0));
  }

  read(tr: TimeRange): VisArray | null {
    if (this.curr.timeRange.overlapsWith(tr)) return this.curr;
    return null;
  }
}
