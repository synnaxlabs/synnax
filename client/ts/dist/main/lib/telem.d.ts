/** Represents a nanosecond precision UTC timestamp. */
export declare class TimeStamp extends Number {
    constructor(value: UnparsedTimeStamp);
    /**
     * Checks if the TimeStamp is equal to another TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamps are equal, false otherwise.
     */
    equals(other: UnparsedTimeStamp): boolean;
    /**
     * Creates a TimeSpan representing the duration between the two timestamps.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns A TimeSpan representing the duration between the two timestamps.
     *   The span is guaranteed to be positive.
     */
    span(other: UnparsedTimeStamp): TimeSpan;
    /**
     * Creates a TimeRange spanning the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns A TimeRange spanning the given TimeStamp that is guaranteed to be
     *   valid, regardless of the TimeStamp order.
     */
    range(other: UnparsedTimeStamp): TimeRange;
    /**
     * Creates a TimeRange starting at the TimeStamp and spanning the given
     * TimeSpan.
     *
     * @param other - The TimeSpan to span.
     * @returns A TimeRange starting at the TimeStamp and spanning the given
     *   TimeSpan. The TimeRange is guaranteed to be valid.
     */
    spanRange(other: UnparsedTimeSpan): TimeRange;
    /**
     * Checks if the TimeStamp represents the unix epoch.
     *
     * @returns True if the TimeStamp represents the unix epoch, false otherwise.
     */
    isZero(): boolean;
    /**
     * Checks if the TimeStamp is after the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is after the given TimeStamp, false
     *   otherwise.
     */
    after(other: UnparsedTimeStamp): boolean;
    /**
     * Checks if the TimeStamp is after or equal to the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is after or equal to the given TimeStamp,
     *   false otherwise.
     */
    afterEq(other: UnparsedTimeStamp): boolean;
    /**
     * Checks if the TimeStamp is before the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is before the given TimeStamp, false
     *   otherwise.
     */
    before(other: UnparsedTimeStamp): boolean;
    /**
     * Checks if TimeStamp is before or equal to the current timestamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if TimeStamp is before or equal to the current timestamp,
     *   false otherwise.
     */
    beforeEq(other: UnparsedTimeStamp): boolean;
    /**
     * Adds a TimeSpan to the TimeStamp.
     *
     * @param span - The TimeSpan to add.
     * @returns A new TimeStamp representing the sum of the TimeStamp and
     *   TimeSpan.
     */
    add(span: UnparsedTimeSpan): TimeStamp;
    /**
     * Subtracts a TimeSpan from the TimeStamp.
     *
     * @param span - The TimeSpan to subtract.
     * @returns A new TimeStamp representing the difference of the TimeStamp and
     *   TimeSpan.
     */
    sub(span: UnparsedTimeSpan): TimeStamp;
    /** The maximum possible value for a timestamp */
    static readonly Max: TimeStamp;
    /** The minimum possible value for a timestamp */
    static readonly Min: TimeStamp;
    /** The unix epoch */
    static readonly Zero: TimeStamp;
}
/** TimeSpan represents a nanosecond precision duration. */
export declare class TimeSpan extends Number {
    constructor(value: UnparsedTimeSpan);
    /** @returns The number of seconds in the TimeSpan. */
    seconds(): number;
    /** @returns The number of milliseconds in the TimeSpan. */
    milliseconds(): number;
    /**
     * Checks if the TimeSpan represents a zero duration.
     *
     * @returns True if the TimeSpan represents a zero duration, false otherwise.
     */
    isZero(): boolean;
    /**
     * Checks if the TimeSpan is equal to another TimeSpan.
     *
     * @returns True if the TimeSpans are equal, false otherwise.
     */
    equals(other: UnparsedTimeSpan): boolean;
    /**
     * Adds a TimeSpan to the TimeSpan.
     *
     * @returns A new TimeSpan representing the sum of the two TimeSpans.
     */
    add(other: UnparsedTimeSpan): TimeSpan;
    /**
     * Creates a TimeSpan representing the duration between the two timestamps.
     *
     * @param other
     */
    sub(other: UnparsedTimeSpan): TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of nanoseconds.
     *
     * @param value - The number of nanoseconds.
     * @returns A TimeSpan representing the given number of nanoseconds.
     */
    static Nanoseconds(value?: UnparsedTimeSpan): TimeSpan;
    /** A nanosecond. */
    static readonly Nanosecond: TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of microseconds.
     *
     * @param value - The number of microseconds.
     * @returns A TimeSpan representing the given number of microseconds.
     */
    static Microseconds(value?: UnparsedTimeStamp): TimeSpan;
    /** A microsecond. */
    static readonly Microsecond: TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of milliseconds.
     *
     * @param value - The number of milliseconds.
     * @returns A TimeSpan representing the given number of milliseconds.
     */
    static Milliseconds(value?: UnparsedTimeStamp): TimeSpan;
    /** A millisecond. */
    static readonly Millisecond: TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of seconds.
     *
     * @param value - The number of seconds.
     * @returns A TimeSpan representing the given number of seconds.
     */
    static Seconds(value?: UnparsedTimeStamp): TimeSpan;
    /** A second. */
    static readonly Second: TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of minutes.
     *
     * @param value - The number of minutes.
     * @returns A TimeSpan representing the given number of minutes.
     */
    static Minutes(value?: UnparsedTimeStamp): TimeSpan;
    /** A minute. */
    static readonly Minute: TimeSpan;
    /**
     * Creates a TimeSpan representing the given number of hours.
     *
     * @param value - The number of hours.
     * @returns A TimeSpan representing the given number of hours.
     */
    static Hours(value?: UnparsedTimeStamp): TimeSpan;
    /** Represents an hour. */
    static readonly Hour: TimeSpan;
    /** The maximum possible value for a TimeSpan. */
    static readonly Max: TimeSpan;
    /** The minimum possible value for a TimeSpan. */
    static readonly Min: TimeSpan;
    /** The zero value for a TimeSpan. */
    static readonly Zero: TimeSpan;
}
/** Rate represents a data rate in Hz. */
export declare class Rate extends Number {
    constructor(value: UnparsedRate);
    /** @returns The number of seconds in the Rate. */
    equals(other: UnparsedRate): boolean;
    /**
     * Calculates the period of the Rate as a TimeSpan.
     *
     * @returns A TimeSpan representing the period of the Rate.
     */
    period(): TimeSpan;
    /**
     * Calculates the number of samples in the given TimeSpan at this rate.
     *
     * @param duration - The duration to calculate the sample count from.
     * @returns The number of samples in the given TimeSpan at this rate.
     */
    sampleCount(duration: UnparsedTimeSpan): number;
    /**
     * Calculates the number of bytes in the given TimeSpan at this rate.
     *
     * @param span - The duration to calculate the byte count from.
     * @param density - The density of the data in bytes per sample.
     * @returns The number of bytes in the given TimeSpan at this rate.
     */
    byteCount(span: UnparsedTimeSpan, density: UnparsedDensity): number;
    /**
     * Calculates a TimeSpan given the number of samples at this rate.
     *
     * @param sampleCount - The number of samples in the span.
     * @returns A TimeSpan that corresponds to the given number of samples.
     */
    span(sampleCount: number): TimeSpan;
    /**
     * Calculates a TimeSpan given the number of bytes at this rate.
     *
     * @param size - The number of bytes in the span.
     * @param density - The density of the data in bytes per sample.
     * @returns A TimeSpan that corresponds to the given number of bytes.
     */
    byteSpan(size: Size, density: UnparsedDensity): TimeSpan;
    /**
     * Creates a Rate representing the given number of Hz.
     *
     * @param value - The number of Hz.
     * @returns A Rate representing the given number of Hz.
     */
    static Hz(value: number): Rate;
    /**
     * Creates a Rate representing the given number of kHz.
     *
     * @param value - The number of kHz.
     * @returns A Rate representing the given number of kHz.
     */
    static KHz(value: number): Rate;
}
/** Density represents the number of bytes in a value. */
export declare class Density extends Number {
    /**
     * Creates a Density representing the given number of bytes per value.
     *
     * @class
     * @param value - The number of bytes per value.
     * @returns A Density representing the given number of bytes per value.
     */
    constructor(value: UnparsedDensity);
    /** Represents an Unknown/Invalid Density. */
    static readonly Unknown: Density;
    /** Represents a Density of 64 bits per value. */
    static readonly Bit64: Density;
    /** Represents a Density of 32 bits per value. */
    static readonly Bit32: Density;
    /** Represents a Density of 16 bits per value. */
    static readonly Bit16: Density;
    /** Represents a Density of 8 bits per value. */
    static readonly Bit8: Density;
}
/**
 * TimeRange represents a range of time between two TimeStamps. It's important
 * to note that the start of the range is inclusive, while the end of the range
 * is exclusive.
 *
 * @property start - A TimeStamp representing the start of the range.
 * @property end - A Timestamp representing the end of the range.
 */
export declare class TimeRange {
    start: TimeStamp;
    end: TimeStamp;
    /**
     * Creates a TimeRange from the given start and end TimeStamps.
     *
     * @param start - A TimeStamp representing the start of the range.
     * @param end - A TimeStamp representing the end of the range.
     */
    constructor(start: UnparsedTimeStamp, end: UnparsedTimeStamp);
    /** @returns The TimeSpan occupied by the TimeRange. */
    span(): TimeSpan;
    /**
     * Checks if the timestamp is valid i.e. the start is before the end.
     *
     * @returns True if the TimeRange is valid.
     */
    isValid(): boolean;
    /**
     * Makes sure the TimeRange is valid i.e. the start is before the end.
     *
     * @returns A TimeRange that is valid.
     */
    makeValid(): TimeRange;
    /**
     * Checks if the TimeRange has a zero span.
     *
     * @returns True if the TimeRange has a zero span.
     */
    isZero(): boolean;
    /**
     * Creates a new TimeRange with the start and end swapped.
     *
     * @returns A TimeRange with the start and end swapped.
     */
    swap(): TimeRange;
    /**
     * Checks if the TimeRange is equal to the given TimeRange.
     *
     * @param other - The TimeRange to compare to.
     * @returns True if the TimeRange is equal to the given TimeRange.
     */
    equals(other: TimeRange): boolean;
    static readonly Max: TimeRange;
}
/** DataType is a string that represents a data type. */
export declare class DataType extends String {
    constructor(value: UnparsedDataType);
    get arrayConstructor(): TypedArrayConstructor;
    get string(): string;
    get density(): Density;
    checkArray(array: TypedArray): boolean;
    toJSON(): string;
    /** Represents an Unknown/Invalid DataType. */
    static readonly Unknown: DataType;
    /** Represents a 64-bit floating point value. */
    static readonly Float64: DataType;
    /** Represents a 32-bit floating point value. */
    static readonly Float32: DataType;
    /** Represents a 64-bit signed integer value. */
    static readonly Int64: DataType;
    /** Represents a 32-bit signed integer value. */
    static readonly Int32: DataType;
    /** Represents a 16-bit signed integer value. */
    static readonly Int16: DataType;
    /** Represents a 8-bit signed integer value. */
    static readonly Int8: DataType;
    /** Represents a 64-bit unsigned integer value. */
    static readonly Uint64: DataType;
    /** Represents a 32-bit unsigned integer value. */
    static readonly Uint32: DataType;
    /** Represents a 16-bit unsigned integer value. */
    static readonly Uint16: DataType;
    /** Represents a 8-bit unsigned integer value. */
    static readonly Uint8: DataType;
}
export declare class Size extends Number {
    constructor(value: UnparsedSize);
    largerThan(other: Size): boolean;
    smallerThan(other: Size): boolean;
    static Bytes(value: UnparsedSize): Size;
    static readonly Byte: Size;
    static Kilobytes(value: UnparsedSize): Size;
    static readonly Kilobyte: Size;
    static Megabytes(value: UnparsedSize): Size;
    static readonly Megabyte: Size;
    static Gigabytes(value: UnparsedSize): Size;
    static readonly Gigabyte: Size;
    static Terabytes(value: UnparsedSize): Size;
    static readonly Terabyte: Size;
}
export declare type UnparsedTimeStamp = TimeStamp | TimeSpan | number;
export declare type UnparsedTimeSpan = TimeSpan | number;
export declare type UnparsedRate = Rate | number;
export declare type UnparsedDensity = Density | number;
export declare type UnparsedDataType = DataType | string;
export declare type UnparsedSize = Size | number;
export declare type TypedArray = Uint8Array | Uint16Array | Uint32Array | BigUint64Array | Float32Array | Float64Array | Int8Array | Int16Array | Int32Array | BigInt64Array;
declare type TypedArrayConstructor = Uint8ArrayConstructor | Uint16ArrayConstructor | Uint32ArrayConstructor | BigUint64ArrayConstructor | Float32ArrayConstructor | Float64ArrayConstructor | Int8ArrayConstructor | Int16ArrayConstructor | Int32ArrayConstructor | BigInt64ArrayConstructor;
export {};
