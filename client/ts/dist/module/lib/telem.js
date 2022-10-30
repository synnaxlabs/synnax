import { registerCustomTypeEncoder } from '@synnaxlabs/freighter';
const valueOfEncoder = (value) => value?.valueOf();
/** Represents a nanosecond precision UTC timestamp. */
export class TimeStamp extends Number {
    constructor(value) {
        super(value);
    }
    /**
     * Checks if the TimeStamp is equal to another TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamps are equal, false otherwise.
     */
    equals(other) {
        return this.valueOf() === new TimeStamp(other).valueOf();
    }
    /**
     * Creates a TimeSpan representing the duration between the two timestamps.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns A TimeSpan representing the duration between the two timestamps.
     *   The span is guaranteed to be positive.
     */
    span(other) {
        return this.range(other).span();
    }
    /**
     * Creates a TimeRange spanning the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns A TimeRange spanning the given TimeStamp that is guaranteed to be
     *   valid, regardless of the TimeStamp order.
     */
    range(other) {
        return new TimeRange(this, other).makeValid();
    }
    /**
     * Creates a TimeRange starting at the TimeStamp and spanning the given
     * TimeSpan.
     *
     * @param other - The TimeSpan to span.
     * @returns A TimeRange starting at the TimeStamp and spanning the given
     *   TimeSpan. The TimeRange is guaranteed to be valid.
     */
    spanRange(other) {
        return this.range(this.add(other)).makeValid();
    }
    /**
     * Checks if the TimeStamp represents the unix epoch.
     *
     * @returns True if the TimeStamp represents the unix epoch, false otherwise.
     */
    isZero() {
        return this.valueOf() === 0;
    }
    /**
     * Checks if the TimeStamp is after the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is after the given TimeStamp, false
     *   otherwise.
     */
    after(other) {
        return this.valueOf() > new TimeStamp(other).valueOf();
    }
    /**
     * Checks if the TimeStamp is after or equal to the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is after or equal to the given TimeStamp,
     *   false otherwise.
     */
    afterEq(other) {
        return this.valueOf() >= new TimeStamp(other).valueOf();
    }
    /**
     * Checks if the TimeStamp is before the given TimeStamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if the TimeStamp is before the given TimeStamp, false
     *   otherwise.
     */
    before(other) {
        return this.valueOf() < new TimeStamp(other).valueOf();
    }
    /**
     * Checks if TimeStamp is before or equal to the current timestamp.
     *
     * @param other - The other TimeStamp to compare to.
     * @returns True if TimeStamp is before or equal to the current timestamp,
     *   false otherwise.
     */
    beforeEq(other) {
        return this.valueOf() <= new TimeStamp(other).valueOf();
    }
    /**
     * Adds a TimeSpan to the TimeStamp.
     *
     * @param span - The TimeSpan to add.
     * @returns A new TimeStamp representing the sum of the TimeStamp and
     *   TimeSpan.
     */
    add(span) {
        return new TimeStamp(this.valueOf() + span.valueOf());
    }
    /**
     * Subtracts a TimeSpan from the TimeStamp.
     *
     * @param span - The TimeSpan to subtract.
     * @returns A new TimeStamp representing the difference of the TimeStamp and
     *   TimeSpan.
     */
    sub(span) {
        return new TimeStamp(this.valueOf() - span.valueOf());
    }
    /** The maximum possible value for a timestamp */
    static Max = new TimeStamp(TimeStamp.MAX_VALUE);
    /** The minimum possible value for a timestamp */
    static Min = new TimeStamp(TimeStamp.MIN_VALUE);
    /** The unix epoch */
    static Zero = new TimeStamp(0);
}
/** TimeSpan represents a nanosecond precision duration. */
export class TimeSpan extends Number {
    constructor(value) {
        super(value);
    }
    /** @returns The number of seconds in the TimeSpan. */
    seconds() {
        return this.valueOf() / TimeSpan.Seconds(1).valueOf();
    }
    /** @returns The number of milliseconds in the TimeSpan. */
    milliseconds() {
        return this.valueOf() / TimeSpan.Milliseconds(1).valueOf();
    }
    /**
     * Checks if the TimeSpan represents a zero duration.
     *
     * @returns True if the TimeSpan represents a zero duration, false otherwise.
     */
    isZero() {
        return this.valueOf() === 0;
    }
    /**
     * Checks if the TimeSpan is equal to another TimeSpan.
     *
     * @returns True if the TimeSpans are equal, false otherwise.
     */
    equals(other) {
        return this.valueOf() === new TimeSpan(other).valueOf();
    }
    /**
     * Adds a TimeSpan to the TimeSpan.
     *
     * @returns A new TimeSpan representing the sum of the two TimeSpans.
     */
    add(other) {
        return new TimeSpan(this.valueOf() + new TimeSpan(other).valueOf());
    }
    /**
     * Creates a TimeSpan representing the duration between the two timestamps.
     *
     * @param other
     */
    sub(other) {
        return new TimeSpan(this.valueOf() - new TimeSpan(other).valueOf());
    }
    /**
     * Creates a TimeSpan representing the given number of nanoseconds.
     *
     * @param value - The number of nanoseconds.
     * @returns A TimeSpan representing the given number of nanoseconds.
     */
    static Nanoseconds(value = 1) {
        return new TimeSpan(value);
    }
    /** A nanosecond. */
    static Nanosecond = TimeSpan.Nanoseconds(1);
    /**
     * Creates a TimeSpan representing the given number of microseconds.
     *
     * @param value - The number of microseconds.
     * @returns A TimeSpan representing the given number of microseconds.
     */
    static Microseconds(value = 1) {
        return TimeSpan.Nanoseconds(value.valueOf() * 1000);
    }
    /** A microsecond. */
    static Microsecond = TimeSpan.Microseconds(1);
    /**
     * Creates a TimeSpan representing the given number of milliseconds.
     *
     * @param value - The number of milliseconds.
     * @returns A TimeSpan representing the given number of milliseconds.
     */
    static Milliseconds(value = 1) {
        return TimeSpan.Microseconds(value.valueOf() * 1000);
    }
    /** A millisecond. */
    static Millisecond = TimeSpan.Milliseconds(1);
    /**
     * Creates a TimeSpan representing the given number of seconds.
     *
     * @param value - The number of seconds.
     * @returns A TimeSpan representing the given number of seconds.
     */
    static Seconds(value = 1) {
        return TimeSpan.Milliseconds(value.valueOf() * 1000);
    }
    /** A second. */
    static Second = TimeSpan.Seconds(1);
    /**
     * Creates a TimeSpan representing the given number of minutes.
     *
     * @param value - The number of minutes.
     * @returns A TimeSpan representing the given number of minutes.
     */
    static Minutes(value = 1) {
        return TimeSpan.Seconds(value.valueOf() * 60);
    }
    /** A minute. */
    static Minute = TimeSpan.Minutes(1);
    /**
     * Creates a TimeSpan representing the given number of hours.
     *
     * @param value - The number of hours.
     * @returns A TimeSpan representing the given number of hours.
     */
    static Hours(value = 1) {
        return TimeSpan.Minutes(value.valueOf() * 60);
    }
    /** Represents an hour. */
    static Hour = TimeSpan.Hours(1);
    /** The maximum possible value for a TimeSpan. */
    static Max = new TimeSpan(TimeSpan.MAX_VALUE);
    /** The minimum possible value for a TimeSpan. */
    static Min = new TimeSpan(TimeSpan.MIN_VALUE);
    /** The zero value for a TimeSpan. */
    static Zero = new TimeSpan(0);
}
/** Rate represents a data rate in Hz. */
export class Rate extends Number {
    constructor(value) {
        super(value);
    }
    /** @returns The number of seconds in the Rate. */
    equals(other) {
        return this.valueOf() === new Rate(other).valueOf();
    }
    /**
     * Calculates the period of the Rate as a TimeSpan.
     *
     * @returns A TimeSpan representing the period of the Rate.
     */
    period() {
        return new TimeSpan(TimeSpan.Seconds(this.valueOf()).valueOf());
    }
    /**
     * Calculates the number of samples in the given TimeSpan at this rate.
     *
     * @param duration - The duration to calculate the sample count from.
     * @returns The number of samples in the given TimeSpan at this rate.
     */
    sampleCount(duration) {
        return new TimeSpan(duration).seconds() * this.valueOf();
    }
    /**
     * Calculates the number of bytes in the given TimeSpan at this rate.
     *
     * @param span - The duration to calculate the byte count from.
     * @param density - The density of the data in bytes per sample.
     * @returns The number of bytes in the given TimeSpan at this rate.
     */
    byteCount(span, density) {
        return this.sampleCount(span) * new Density(density).valueOf();
    }
    /**
     * Calculates a TimeSpan given the number of samples at this rate.
     *
     * @param sampleCount - The number of samples in the span.
     * @returns A TimeSpan that corresponds to the given number of samples.
     */
    span(sampleCount) {
        return TimeSpan.Seconds(sampleCount / this.valueOf());
    }
    /**
     * Calculates a TimeSpan given the number of bytes at this rate.
     *
     * @param size - The number of bytes in the span.
     * @param density - The density of the data in bytes per sample.
     * @returns A TimeSpan that corresponds to the given number of bytes.
     */
    byteSpan(size, density) {
        return this.span(size.valueOf() / density.valueOf());
    }
    /**
     * Creates a Rate representing the given number of Hz.
     *
     * @param value - The number of Hz.
     * @returns A Rate representing the given number of Hz.
     */
    static Hz(value) {
        return new Rate(value);
    }
    /**
     * Creates a Rate representing the given number of kHz.
     *
     * @param value - The number of kHz.
     * @returns A Rate representing the given number of kHz.
     */
    static KHz(value) {
        return Rate.Hz(value * 1000);
    }
}
/** Density represents the number of bytes in a value. */
export class Density extends Number {
    /**
     * Creates a Density representing the given number of bytes per value.
     *
     * @class
     * @param value - The number of bytes per value.
     * @returns A Density representing the given number of bytes per value.
     */
    constructor(value) {
        super(value);
    }
    /** Represents an Unknown/Invalid Density. */
    static Unknown = new Density(0);
    /** Represents a Density of 64 bits per value. */
    static Bit64 = new Density(8);
    /** Represents a Density of 32 bits per value. */
    static Bit32 = new Density(4);
    /** Represents a Density of 16 bits per value. */
    static Bit16 = new Density(2);
    /** Represents a Density of 8 bits per value. */
    static Bit8 = new Density(1);
}
/**
 * TimeRange represents a range of time between two TimeStamps. It's important
 * to note that the start of the range is inclusive, while the end of the range
 * is exclusive.
 *
 * @property start - A TimeStamp representing the start of the range.
 * @property end - A Timestamp representing the end of the range.
 */
export class TimeRange {
    start;
    end;
    /**
     * Creates a TimeRange from the given start and end TimeStamps.
     *
     * @param start - A TimeStamp representing the start of the range.
     * @param end - A TimeStamp representing the end of the range.
     */
    constructor(start, end) {
        this.start = new TimeStamp(start);
        this.end = new TimeStamp(end);
    }
    /** @returns The TimeSpan occupied by the TimeRange. */
    span() {
        return new TimeSpan(this.end.valueOf() - this.start.valueOf());
    }
    /**
     * Checks if the timestamp is valid i.e. the start is before the end.
     *
     * @returns True if the TimeRange is valid.
     */
    isValid() {
        return this.start.valueOf() <= this.end.valueOf();
    }
    /**
     * Makes sure the TimeRange is valid i.e. the start is before the end.
     *
     * @returns A TimeRange that is valid.
     */
    makeValid() {
        return this.isValid() ? this : this.swap();
    }
    /**
     * Checks if the TimeRange has a zero span.
     *
     * @returns True if the TimeRange has a zero span.
     */
    isZero() {
        return this.span().isZero();
    }
    /**
     * Creates a new TimeRange with the start and end swapped.
     *
     * @returns A TimeRange with the start and end swapped.
     */
    swap() {
        return new TimeRange(this.end, this.start);
    }
    /**
     * Checks if the TimeRange is equal to the given TimeRange.
     *
     * @param other - The TimeRange to compare to.
     * @returns True if the TimeRange is equal to the given TimeRange.
     */
    equals(other) {
        return this.start.equals(other.start) && this.end.equals(other.end);
    }
    static Max = new TimeRange(TimeStamp.Min, TimeStamp.Max);
}
/** DataType is a string that represents a data type. */
export class DataType extends String {
    constructor(value) {
        if (typeof value === 'string') {
            super(value);
        }
        else {
            super(value.valueOf());
        }
    }
    get arrayConstructor() {
        const v = ARRAY_CONSTRUCTORS.get(this.string);
        if (v === undefined) {
            throw new Error(`Unknown data type: ${this.string}`);
        }
        return v;
    }
    get string() {
        return this.valueOf();
    }
    get density() {
        const v = DATA_TYPE_DENSITIES.get(this.string);
        if (v === undefined) {
            throw new Error(`Unknown data type: ${this.string}`);
        }
        return v;
    }
    checkArray(array) {
        return array.constructor === this.arrayConstructor;
    }
    toJSON() {
        return this.string;
    }
    /** Represents an Unknown/Invalid DataType. */
    static Unknown = new DataType('unknown');
    /** Represents a 64-bit floating point value. */
    static Float64 = new DataType('float64');
    /** Represents a 32-bit floating point value. */
    static Float32 = new DataType('float32');
    /** Represents a 64-bit signed integer value. */
    static Int64 = new DataType('int64');
    /** Represents a 32-bit signed integer value. */
    static Int32 = new DataType('int32');
    /** Represents a 16-bit signed integer value. */
    static Int16 = new DataType('int16');
    /** Represents a 8-bit signed integer value. */
    static Int8 = new DataType('int8');
    /** Represents a 64-bit unsigned integer value. */
    static Uint64 = new DataType('uint64');
    /** Represents a 32-bit unsigned integer value. */
    static Uint32 = new DataType('uint32');
    /** Represents a 16-bit unsigned integer value. */
    static Uint16 = new DataType('uint16');
    /** Represents a 8-bit unsigned integer value. */
    static Uint8 = new DataType('uint8');
}
export class Size extends Number {
    constructor(value) {
        super(value.valueOf());
    }
    largerThan(other) {
        return this.valueOf() > other.valueOf();
    }
    smallerThan(other) {
        return this.valueOf() < other.valueOf();
    }
    static Bytes(value) {
        return new Size(value);
    }
    static Byte = new Size(1);
    static Kilobytes(value) {
        return Size.Bytes(value.valueOf() * 1e3);
    }
    static Kilobyte = Size.Kilobytes(1);
    static Megabytes(value) {
        return Size.Kilobytes(value.valueOf() * 1e3);
    }
    static Megabyte = Size.Megabytes(1);
    static Gigabytes(value) {
        return Size.Megabytes(value.valueOf() * 1e3);
    }
    static Gigabyte = Size.Gigabytes(1);
    static Terabytes(value) {
        return Size.Gigabytes(value.valueOf() * 1e3);
    }
    static Terabyte = Size.Terabytes(1);
}
registerCustomTypeEncoder({ Class: TimeStamp, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: TimeSpan, write: valueOfEncoder });
registerCustomTypeEncoder({
    Class: DataType,
    write: (v) => v.string,
});
registerCustomTypeEncoder({ Class: Rate, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: Density, write: valueOfEncoder });
const ARRAY_CONSTRUCTORS = new Map([
    [DataType.Uint8.string, Uint8Array],
    [DataType.Uint16.string, Uint16Array],
    [DataType.Uint32.string, Uint32Array],
    [DataType.Uint64.string, BigUint64Array],
    [DataType.Float32.string, Float32Array],
    [DataType.Float64.string, Float64Array],
    [DataType.Int8.string, Int8Array],
    [DataType.Int16.string, Int16Array],
    [DataType.Int32.string, Int32Array],
    [DataType.Int64.string, BigInt64Array],
]);
const DATA_TYPE_DENSITIES = new Map([
    [DataType.Uint8.string, Density.Bit8],
    [DataType.Uint16.string, Density.Bit16],
    [DataType.Uint32.string, Density.Bit32],
    [DataType.Uint64.string, Density.Bit64],
    [DataType.Float32.string, Density.Bit32],
    [DataType.Float64.string, Density.Bit64],
    [DataType.Int8.string, Density.Bit8],
    [DataType.Int16.string, Density.Bit16],
    [DataType.Int32.string, Density.Bit32],
    [DataType.Int64.string, Density.Bit64],
]);
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidGVsZW0uanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL3RlbGVtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSx5QkFBeUIsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBRWxFLE1BQU0sY0FBYyxHQUFHLENBQUMsS0FBYyxFQUFXLEVBQUUsQ0FBQyxLQUFLLEVBQUUsT0FBTyxFQUFFLENBQUM7QUFFckUsdURBQXVEO0FBQ3ZELE1BQU0sT0FBTyxTQUFVLFNBQVEsTUFBTTtJQUNuQyxZQUFZLEtBQXdCO1FBQ2xDLEtBQUssQ0FBQyxLQUFLLENBQUMsQ0FBQztJQUNmLENBQUM7SUFFRDs7Ozs7T0FLRztJQUNILE1BQU0sQ0FBQyxLQUF3QjtRQUM3QixPQUFPLElBQUksQ0FBQyxPQUFPLEVBQUUsS0FBSyxJQUFJLFNBQVMsQ0FBQyxLQUFLLENBQUMsQ0FBQyxPQUFPLEVBQUUsQ0FBQztJQUMzRCxDQUFDO0lBRUQ7Ozs7OztPQU1HO0lBQ0gsSUFBSSxDQUFDLEtBQXdCO1FBQzNCLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxLQUFLLENBQUMsQ0FBQyxJQUFJLEVBQUUsQ0FBQztJQUNsQyxDQUFDO0lBRUQ7Ozs7OztPQU1HO0lBQ0gsS0FBSyxDQUFDLEtBQXdCO1FBQzVCLE9BQU8sSUFBSSxTQUFTLENBQUMsSUFBSSxFQUFFLEtBQUssQ0FBQyxDQUFDLFNBQVMsRUFBRSxDQUFDO0lBQ2hELENBQUM7SUFFRDs7Ozs7OztPQU9HO0lBQ0gsU0FBUyxDQUFDLEtBQXVCO1FBQy9CLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsU0FBUyxFQUFFLENBQUM7SUFDakQsQ0FBQztJQUVEOzs7O09BSUc7SUFDSCxNQUFNO1FBQ0osT0FBTyxJQUFJLENBQUMsT0FBTyxFQUFFLEtBQUssQ0FBQyxDQUFDO0lBQzlCLENBQUM7SUFFRDs7Ozs7O09B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