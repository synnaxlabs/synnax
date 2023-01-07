import {
  convertDataType,
  DataType,
  NativeTypedArray,
  Size,
  TimeRange,
  UnparsedDataType,
} from "./telem";

import { validateFieldNotNull, ValidationError } from "@/errors";

/** A strongly typed array of telemetry samples. */
export class TArray {
  readonly dataType: DataType;
  private readonly _data: ArrayBufferLike;
  readonly _timeRange?: TimeRange;

  constructor(
    data: ArrayBufferLike | NativeTypedArray,
    dataType?: UnparsedDataType,
    timeRange?: TimeRange
  ) {
    if (
      dataType == null &&
      !(data instanceof ArrayBuffer) &&
      !(data instanceof SharedArrayBuffer)
    ) {
      this.dataType = new DataType(data);
    } else if (dataType != null) {
      this.dataType = new DataType(dataType);
    } else {
      throw new ValidationError(
        "must provide a data type when constructing a TArray from a buffer"
      );
    }
    this._data = data;
    this._timeRange = timeRange;
  }

  /** @returns the underlying buffer backing this array. */
  get buffer(): ArrayBufferLike {
    return this._data;
  }

  /** @returns a native typed array with the proper data type. */
  get data(): NativeTypedArray {
    validateFieldNotNull("dataType", this._data);
    return new this.dataType.Array(this._data);
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    validateFieldNotNull("_timeRange", this._timeRange);
    return this._timeRange as TimeRange;
  }

  /** @returns the size of the underlying buffer in bytes. */
  get size(): Size {
    return new Size(this.buffer.byteLength);
  }

  /** @returns the number of samples in this array. */
  get length(): number {
    return this.dataType.density.length(this.size);
  }

  /**
   * Creates a new array with a different data type.
   * @param target the data type to convert to.
   * @param offset an offset to apply to each sample. This can help with precision
   * issues when converting between data types.
   *
   * WARNING: This method is expensive and copies the entire underlying array. There
   * also may be untimely precision issues when converting between data types.
   */
  convert(target: DataType, offset: number | bigint = 0): TArray {
    if (this.dataType.equals(target)) return this;
    const data = new target.Array(this.length);
    for (let i = 0; i < this.length; i++) {
      data[i] = convertDataType(this.dataType, target, this.data[i], offset);
    }
    return new TArray(data.buffer, target, this._timeRange);
  }
}
