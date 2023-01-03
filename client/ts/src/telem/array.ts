import { DataType, TimeRange, NativeTypedArray, UnparsedDataType } from "./telem";

import { ValidationError } from "@/errors";

/**
 * A TArray is a typed array of telemetry backed by a continuous buffer of data as well
 * as an optional time range.
 */
export class TypedArray {
  readonly dataType: DataType;
  private readonly _data: ArrayBufferLike;
  private readonly _timeRange?: TimeRange;

  constructor(
    dataType: UnparsedDataType,
    data: ArrayBufferLike,
    timeRange?: TimeRange
  ) {
    this.dataType = new DataType(dataType);
    this._data = data;
    this._timeRange = timeRange;
  }

  get buffer(): ArrayBufferLike {
    if (this._data == null) throw new ValidationError("data is null");
    return this._data;
  }

  get data(): NativeTypedArray {
    if (this._data == null) throw new ValidationError("data is null");
    return new this.dataType.Array(this._data);
  }

  get timeRange(): TimeRange {
    if (this._timeRange == null) throw new ValidationError("timeRange is null");
    return this._timeRange;
  }

  get length(): number {
    return this.data.length;
  }
}
