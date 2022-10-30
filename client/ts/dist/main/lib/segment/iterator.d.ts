import { StreamClient } from '@synnaxlabs/freighter';
import Registry from '../channel/registry';
import { TimeRange } from '../telem';
import { SegmentPayload } from './payload';
import TypedSegment from './typed';
/**
 * Used to iterate over a clusters telemetry in time-order. It should not be
 * instantiated directly, and should instead be instantiated via the SegmentClient.
 *
 * Using an iterator is ideal when querying/processing large ranges of data, but
 * is relatively complex and difficult to use. If you're looking to retrieve
 *  telemetry between two timestamps, see the SegmentClient.read method.
 */
export declare class CoreIterator {
    private static ENDPOINT;
    private client;
    private stream;
    private readonly aggregate;
    values: SegmentPayload[];
    constructor(client: StreamClient, aggregate?: boolean);
    /**
     * Opens the iterator, configuring it to iterate over the telemetry in the
     * channels with the given keys within the provided time range.
     *
     * @param tr - The time range to iterate over.
     * @param keys - The keys of the channels to iterate over.
     */
    open(tr: TimeRange, keys: string[]): Promise<void>;
    /**
     * Reads the next segment for each channel in the iterator.
     *
     * @returns false if the next segment can't be found for one or more channels or
     * the iterator has accumulated an error.
     */
    next(): Promise<boolean>;
    /**
     * Reads the previous segment for each channel in the iterator.
     *
     * @returns false if the next segment can't be found for one or more channels or
     * the iterator has accumulated an error.
     */
    prev(): Promise<boolean>;
    /**
     * Seeks to the beginning of the time range and reads the first segment of each
     * channel in the iterator.
     *
     * @returns false if no segments exists in the time range for a particular channel
     * or the iterator has accumulated an error.
     */
    first(): Promise<boolean>;
    /**
     * Seeks to the end of the time range and reads the last segment of each channel
     * in the iterator.
     *
     * @returns false if no segments exists in the time range for a particular channel,
     * or the iterator has accumulated an error.
     */
    last(): Promise<boolean>;
    /**
     * Reads the next time span of telemetry for each channel in the iterator.
     *
     * @returns false if a segment satisfying the request can't be found for a
     * particular channel or the iterator has accumulated an error.
     */
    nextSpan(span: number): Promise<boolean>;
    /**
     * Reads the previous time span of telemetry for each channel in the iterator.
     *
     * @returns false if a segment satisfying the request can't be found for a particular
     * channel or the iterator has accumulated an error.
     */
    prevSpan(span: number): Promise<boolean>;
    /**
     * Seeks the iterator to the start of the time range and reads the telemetry within
     * the range for each channel.
     *
     * @returns: False if a segment satisfying the request can't be found for a particular
     * channel or the iterator has accumulated an error.
     */
    nextRange(range: TimeRange): Promise<boolean>;
    /**
     * Seeks the iterator to the first segment in the time range, but does not read
     * it. Also invalidates the iterator. The iterator will not be considered valid
     * until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    seekFirst(): Promise<boolean>;
    /** Seeks the iterator to the last segment in the time range, but does not read it.
     * Also invalidates the iterator. The iterator will not be considered valid
     * until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    seekLast(): Promise<boolean>;
    /**
     * Seeks the iterator to the first segment whose start is less than or equal to
     * the provided timestamp. Also invalidates the iterator. The iterator will not be
     * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    seekLT(stamp: number): Promise<boolean>;
    /**
     * Seeks the iterator to the first segment whose start is greater than or equal to
     * the provided timestamp. Also invalidates the iterator. The iterator will not be
     * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    seekGE(stamp: number): Promise<boolean>;
    /**
     * @returns true if the iterator value contains a valid segment, and fale otherwise.
     * valid most commonly returns false when the iterator is exhausted or has
     * accumulated an error.
     */
    valid(): Promise<boolean>;
    /**
     * Closes the iterator. An iterator MUST be closed after use, and this method
     * should probably be placed in a 'finally' block. If the iterator is not closed,
     * it may leak resources.
     */
    close(): Promise<void>;
    private execute;
}
export declare class TypedIterator extends CoreIterator {
    channels: Registry;
    constructor(client: StreamClient, channels: Registry, aggregate?: boolean);
    value(): Promise<Record<string, TypedSegment>>;
}
