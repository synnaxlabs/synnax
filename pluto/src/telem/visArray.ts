import { LazyArray, TimeRange } from "@synnaxlabs/x";

export class VisArray {
  readonly gl: WebGLBuffer;
  readonly arr: LazyArray;
  demand: number;
  private _length: number | null;

  constructor(gl: WebGLBuffer, arr: LazyArray) {
    this.demand = 0;
    this.gl = gl;
    this.arr = arr;
    this._length = null;
  }

  get timeRange(): TimeRange {
    return this.arr.timeRange;
  }

  get available(): number {
    return this.arr.length - this.length;
  }

  get length(): number {
    if (this._length == null) return this.arr.length;
    return this._length;
  }

  set length(length: number) {
    this._length = length;
  }

  acquire(): void {
    this.demand++;
  }

  release(): void {
    this.demand--;
  }
}
