import { StreamClient } from '@synnaxlabs/freighter';
import ChannelRegistry from '../channel/registry';
import { TypedArray, UnparsedTimeStamp } from '../telem';
import { SegmentPayload } from './payload';
/**
 * CoreWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
export declare class CoreWriter {
    private static ENDPOINT;
    private client;
    private stream;
    private keys;
    constructor(client: StreamClient);
    /**
     * Opens the writer, acquiring an exclusive lock on the given channels for
     * the duration of the writer's lifetime. open must be called before any other
     * writer methods.
     *
     * @param keys - A list of keys representing the channels the writer will write
     * to.
     */
    open(keys: string[]): Promise<void>;
    /**
     * Validates and writes the given segments to the database. The provided segments
     * must:
     *
     *   1. Be in time order (on a per-channel basis)
     *   2. Have channel keys in the set of keys this writer was opened with.
     *   3. Have non-zero length data with the correct data type.
     *
     * @param segments - A list of segments to write to the database.
     * @returns false if the writer has accumulated an error. In this case,
     * the caller should stop executing requests and close the writer.
     */
    write(segments: SegmentPayload[]): Promise<boolean>;
    /**
     * Closes the writer, raising any accumulated error encountered during operation.
     * A writer MUST be closed after use, and this method should probably be placed
     * in a 'finally' block. If the writer is not closed, the database will not release
     * the exclusive lock on the channels, preventing any other callers from
     * writing to them. It also might leak resources and threads.
     */
    close(): Promise<void>;
    private checkKeys;
}
/**
 * TypedWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
export declare class TypedWriter {
    private core;
    private splitter;
    private channels;
    private scalarTypeValidator;
    private contiguityValidator;
    constructor(client: StreamClient, channels: ChannelRegistry);
    /**
     * Opens the writer, acquiring an exclusive lock on the given channels for
     * the duration of the writer's lifetime. open must be called before any other
     * writer methods.
     *
     * @param keys - A list of keys representing the channels the writer will write
     * to.
     */
    open(keys: string[]): Promise<void>;
    /**
     * Writes the given telemetry to the database.
     *
     * @param to - They key of the channel to write to. This must be in the set of
     * keys this writer was opened with.
     * @param start - The start time of the telemetry. This must be equal to
     * the end of the previous segment written to the channel (unless it's the first
     * write to that channel).
     * @param data - The telemetry to write. This must be a valid type for the channel.
     * @returns false if the writer has accumulated an error. In this case,
     * the caller should stop executing requests and close the writer.
     */
    write(to: string, start: UnparsedTimeStamp, data: TypedArray): Promise<boolean>;
    /**
     * Closes the writer, raising any accumulated error encountered during operation.
     * A writer MUST be closed after use, and this method should probably be placed
     * in a 'finally' block. If the writer is not closed, the database will not release
     * the exclusive lock on the channels, preventing any other callers from
     * writing to them. It also might leak resources and threads.
     */
    close(): Promise<void>;
}
