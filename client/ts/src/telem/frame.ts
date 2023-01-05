import { TypedArray } from "./array";

export class Frame {
  arrays: TypedArray[];

  constructor(arrays: TypedArray[] = []) {
    this.arrays = arrays;
  }

  get even(): boolean {
    const first = this.arrays[0];
    return this.arrays
      .slice(1)
      .every(
        (array) =>
          array.length === first.length && array.timeRange.equals(first.timeRange)
      );
  }
}
