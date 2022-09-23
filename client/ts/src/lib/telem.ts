export class TimeStamp extends Number {
  constructor(value: UnparsedTimeStamp) {
    super(value);
  }

  span(other: TimeStamp): TimeStamp {
    return new TimeStamp(this.valueOf() - other.valueOf());
  }

  isZero(): boolean {
    return this.valueOf() === 0;
  }

  after(other: TimeStamp): boolean {
    return this.valueOf() > other.valueOf();
  }

  afterEq(other: TimeStamp): boolean {
    return this.valueOf() >= other.valueOf();
  }

  before(other: TimeStamp): boolean {
    return this.valueOf() < other.valueOf();
  }

  beforeEq(other: TimeStamp): boolean {
    return this.valueOf() <= other.valueOf();
  }
}

export class TimeSpan extends Number {
  constructor(value: UnparsedTimeSpan) {
    super(value);
  }

  seconds(): number {
    return this.valueOf() / TimeSpan.Second(1).valueOf();
  }

  isZero(): boolean {
    return this.valueOf() === 0;
  }

  equals(other: UnparsedTimeSpan): boolean {
    return this.valueOf() === new TimeSpan(other).valueOf();
  }

  add(other: UnparsedTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() + new TimeSpan(other).valueOf());
  }

  sub(other: UnparsedTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() - new TimeSpan(other).valueOf());
  }

  static Nanosecond(value: UnparsedTimeSpan): TimeSpan {
    return new TimeSpan(value);
  }

  static Microsecond(value: number): TimeSpan {
    return TimeSpan.Nanosecond(value * 1000);
  }

  static Millisecond(value: number): TimeSpan {
    return TimeSpan.Microsecond(value * 1000);
  }

  static Second(value: number): TimeSpan {
    return TimeSpan.Millisecond(value * 1000);
  }

  static Minute(value: number): TimeSpan {
    return TimeSpan.Second(value * 60);
  }

  static Hour(value: number): TimeSpan {
    return TimeSpan.Minute(value * 60);
  }
}

export class Rate extends Number {
  constructor(value: UnparsedRate) {
    super(value);
  }

  static Hz(value: number): Rate {
    return new Rate(value);
  }

  static KHz(value: number): Rate {
    return Rate.Hz(value * 1000);
  }

  period(): TimeSpan {
    return new TimeSpan(1 / TimeSpan.Second(this.valueOf()).valueOf());
  }

  sampleCount(duration: TimeSpan): number {
    return duration.seconds() * this.valueOf();
  }

  byteSize(timeSpan: TimeSpan): number {
    return this.sampleCount(timeSpan) * 4;
  }

  span(sampleCount: number): TimeSpan {
    return TimeSpan.Second(sampleCount / this.valueOf());
  }

  sizeSpan(byteSize: number): TimeSpan {
    return this.span(byteSize / 4);
  }
}

export class Density extends Number {
  constructor(value: UnparsedDensity) {
    super(value);
  }

  static readonly Unknown = new Density(0);
  static readonly Bit64 = new Density(64);
  static readonly Bit32 = new Density(32);
  static readonly Bit16 = new Density(16);
  static readonly Bit8 = new Density(8);
}

export class TimeRange {
  start: TimeStamp;
  end: TimeStamp;

  constructor(start: UnparsedTimeStamp, end: UnparsedTimeStamp) {
    this.start = new TimeStamp(start);
    this.end = new TimeStamp(end);
  }
}

export class DataType extends String {
  constructor(value: UnparsedDataType) {
    super(value);
  }

  static readonly Unknown = new DataType('unknown');
  static readonly Float64 = new DataType('float64');
  static readonly Float32 = new DataType('float32');
  static readonly Int64 = new DataType('int64');
  static readonly Int32 = new DataType('int32');
  static readonly Int16 = new DataType('int16');
  static readonly Int8 = new DataType('int8');
  static readonly Uint64 = new DataType('uint64');
  static readonly Uint32 = new DataType('uint32');
  static readonly Uint16 = new DataType('uint16');
  static readonly Uint8 = new DataType('uint8');
  static readonly Bool = new DataType('bool');
  static readonly String = new DataType('string');
}

type UnparsedTimeStamp = TimeStamp | number | Number;
type UnparsedTimeSpan = TimeSpan | number | Number;
type UnparsedRate = Rate | number | Number;
type UnparsedDensity = Density | number | Number;
type UnparsedDataType = DataType | string | String;
