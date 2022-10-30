"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Size = exports.DataType = exports.TimeRange = exports.Density = exports.Rate = exports.TimeSpan = exports.TimeStamp = void 0;
const freighter_1 = require("@synnaxlabs/freighter");
const valueOfEncoder = (value) => value === null || value === void 0 ? void 0 : value.valueOf();
/** Represents a nanosecond precision UTC timestamp. */
class TimeStamp extends Number {
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
}
exports.TimeStamp = TimeStamp;
/** The maximum possible value for a timestamp */
TimeStamp.Max = new TimeStamp(TimeStamp.MAX_VALUE);
/** The minimum possible value for a timestamp */
TimeStamp.Min = new TimeStamp(TimeStamp.MIN_VALUE);
/** The unix epoch */
TimeStamp.Zero = new TimeStamp(0);
/** TimeSpan represents a nanosecond precision duration. */
class TimeSpan extends Number {
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
    /**
     * Creates a TimeSpan representing the given number of microseconds.
     *
     * @param value - The number of microseconds.
     * @returns A TimeSpan representing the given number of microseconds.
     */
    static Microseconds(value = 1) {
        return TimeSpan.Nanoseconds(value.valueOf() * 1000);
    }
    /**
     * Creates a TimeSpan representing the given number of milliseconds.
     *
     * @param value - The number of milliseconds.
     * @returns A TimeSpan representing the given number of milliseconds.
     */
    static Milliseconds(value = 1) {
        return TimeSpan.Microseconds(value.valueOf() * 1000);
    }
    /**
     * Creates a TimeSpan representing the given number of seconds.
     *
     * @param value - The number of seconds.
     * @returns A TimeSpan representing the given number of seconds.
     */
    static Seconds(value = 1) {
        return TimeSpan.Milliseconds(value.valueOf() * 1000);
    }
    /**
     * Creates a TimeSpan representing the given number of minutes.
     *
     * @param value - The number of minutes.
     * @returns A TimeSpan representing the given number of minutes.
     */
    static Minutes(value = 1) {
        return TimeSpan.Seconds(value.valueOf() * 60);
    }
    /**
     * Creates a TimeSpan representing the given number of hours.
     *
     * @param value - The number of hours.
     * @returns A TimeSpan representing the given number of hours.
     */
    static Hours(value = 1) {
        return TimeSpan.Minutes(value.valueOf() * 60);
    }
}
exports.TimeSpan = TimeSpan;
/** A nanosecond. */
TimeSpan.Nanosecond = TimeSpan.Nanoseconds(1);
/** A microsecond. */
TimeSpan.Microsecond = TimeSpan.Microseconds(1);
/** A millisecond. */
TimeSpan.Millisecond = TimeSpan.Milliseconds(1);
/** A second. */
TimeSpan.Second = TimeSpan.Seconds(1);
/** A minute. */
TimeSpan.Minute = TimeSpan.Minutes(1);
/** Represents an hour. */
TimeSpan.Hour = TimeSpan.Hours(1);
/** The maximum possible value for a TimeSpan. */
TimeSpan.Max = new TimeSpan(TimeSpan.MAX_VALUE);
/** The minimum possible value for a TimeSpan. */
TimeSpan.Min = new TimeSpan(TimeSpan.MIN_VALUE);
/** The zero value for a TimeSpan. */
TimeSpan.Zero = new TimeSpan(0);
/** Rate represents a data rate in Hz. */
class Rate extends Number {
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
exports.Rate = Rate;
/** Density represents the number of bytes in a value. */
class Density extends Number {
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
}
exports.Density = Density;
/** Represents an Unknown/Invalid Density. */
Density.Unknown = new Density(0);
/** Represents a Density of 64 bits per value. */
Density.Bit64 = new Density(8);
/** Represents a Density of 32 bits per value. */
Density.Bit32 = new Density(4);
/** Represents a Density of 16 bits per value. */
Density.Bit16 = new Density(2);
/** Represents a Density of 8 bits per value. */
Density.Bit8 = new Density(1);
/**
 * TimeRange represents a range of time between two TimeStamps. It's important
 * to note that the start of the range is inclusive, while the end of the range
 * is exclusive.
 *
 * @property start - A TimeStamp representing the start of the range.
 * @property end - A Timestamp representing the end of the range.
 */
class TimeRange {
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
}
exports.TimeRange = TimeRange;
TimeRange.Max = new TimeRange(TimeStamp.Min, TimeStamp.Max);
/** DataType is a string that represents a data type. */
class DataType extends String {
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
}
exports.DataType = DataType;
/** Represents an Unknown/Invalid DataType. */
DataType.Unknown = new DataType('unknown');
/** Represents a 64-bit floating point value. */
DataType.Float64 = new DataType('float64');
/** Represents a 32-bit floating point value. */
DataType.Float32 = new DataType('float32');
/** Represents a 64-bit signed integer value. */
DataType.Int64 = new DataType('int64');
/** Represents a 32-bit signed integer value. */
DataType.Int32 = new DataType('int32');
/** Represents a 16-bit signed integer value. */
DataType.Int16 = new DataType('int16');
/** Represents a 8-bit signed integer value. */
DataType.Int8 = new DataType('int8');
/** Represents a 64-bit unsigned integer value. */
DataType.Uint64 = new DataType('uint64');
/** Represents a 32-bit unsigned integer value. */
DataType.Uint32 = new DataType('uint32');
/** Represents a 16-bit unsigned integer value. */
DataType.Uint16 = new DataType('uint16');
/** Represents a 8-bit unsigned integer value. */
DataType.Uint8 = new DataType('uint8');
class Size extends Number {
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
    static Kilobytes(value) {
        return Size.Bytes(value.valueOf() * 1e3);
    }
    static Megabytes(value) {
        return Size.Kilobytes(value.valueOf() * 1e3);
    }
    static Gigabytes(value) {
        return Size.Megabytes(value.valueOf() * 1e3);
    }
    static Terabytes(value) {
        return Size.Gigabytes(value.valueOf() * 1e3);
    }
}
exports.Size = Size;
Size.Byte = new Size(1);
Size.Kilobyte = Size.Kilobytes(1);
Size.Megabyte = Size.Megabytes(1);
Size.Gigabyte = Size.Gigabytes(1);
Size.Terabyte = Size.Terabytes(1);
(0, freighter_1.registerCustomTypeEncoder)({ Class: TimeStamp, write: valueOfEncoder });
(0, freighter_1.registerCustomTypeEncoder)({ Class: TimeSpan, write: valueOfEncoder });
(0, freighter_1.registerCustomTypeEncoder)({
    Class: DataType,
    write: (v) => v.string,
});
(0, freighter_1.registerCustomTypeEncoder)({ Class: Rate, write: valueOfEncoder });
(0, freighter_1.registerCustomTypeEncoder)({ Class: Density, write: valueOfEncoder });
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidGVsZW0uanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL3RlbGVtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7OztBQUFBLHFEQUFrRTtBQUVsRSxNQUFNLGNBQWMsR0FBRyxDQUFDLEtBQWMsRUFBVyxFQUFFLENBQUMsS0FBSyxhQUFMLEtBQUssdUJBQUwsS0FBSyxDQUFFLE9BQU8sRUFBRSxDQUFDO0FBRXJFLHVEQUF1RDtBQUN2RCxNQUFhLFNBQVUsU0FBUSxNQUFNO0lBQ25DLFlBQVksS0FBd0I7UUFDbEMsS0FBSyxDQUFDLEtBQUssQ0FBQyxDQUFDO0lBQ2YsQ0FBQztJQUVEOzs7OztPQUtHO0lBQ0gsTUFBTSxDQUFDLEtBQXdCO1FBQzdCLE9BQU8sSUFBSSxDQUFDLE9BQU8sRUFBRSxLQUFLLElBQUksU0FBUyxDQUFDLEtBQUssQ0FBQyxDQUFDLE9BQU8sRUFBRSxDQUFDO0lBQzNELENBQUM7SUFFRDs7Ozs7O09BTUc7SUFDSCxJQUFJLENBQUMsS0FBd0I7UUFDM0IsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLEtBQUssQ0FBQyxDQUFDLElBQUksRUFBRSxDQUFDO0lBQ2xDLENBQUM7SUFFRDs7Ozs7O09BTUc7SUFDSCxLQUFLLENBQUMsS0FBd0I7UUFDNUIsT0FBTyxJQUFJLFNBQVMsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUMsU0FBUyxFQUFFLENBQUM7SUFDaEQsQ0FBQztJQUVEOzs7Ozs7O09BT0c7SUFDSCxTQUFTLENBQUMsS0FBdUI7UUFDL0IsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxTQUFTLEVBQUUsQ0FBQztJQUNqRCxDQUFDO0lBRUQ7Ozs7T0FJRztJQUNILE1BQU07UUFDSixPQUFPLElBQUksQ0FBQyxPQUFPLEVBQUUsS0FBSyxDQUFDLENBQUM7SUFDOUIsQ0FBQztJQUVEOzs7Ozs7T0FNRztJQUNILEtBQUss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