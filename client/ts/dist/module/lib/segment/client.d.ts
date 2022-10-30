import Registry from '../channel/registry';
import { TimeRange, TypedArray, UnparsedTimeStamp } from '../telem';
import Transport from '../transport';
import { TypedIterator } from './iterator';
import TypedSegment from './typed';
import { TypedWriter } from './writer';
export default class SegmentClient {
    private transport;
    private channels;
    constructor(transport: Transport, channels: Registry);
    /**
     * Opens a new iterator over the given channels within the provided time range.
     *
     * @param tr - A time range to iterate over.
     * @param keys - A list of channel keys to iterate over.
     * @param aggregate - Whether to accumulate iteration results or reset them
     * on every iterator method call.
     * @returns a new {@link TypedIterator}.
     */
    newIterator(tr: TimeRange, keys: string[], aggregate: boolean): Promise<TypedIterator>;
    /**
     * Opens a new writer on the given channels.
     *
     * @param keys - The keys of the channels to write to. A writer cannot write to
     * a channel that is not in this list. See the {@link TypedWriter} documentation
     * for more information.
     * @returns a new {@link TypedWriter}.
     */
    newWriter(keys: string[]): Promise<TypedWriter>;
    /**
     * Writes telemetry to the given channel starting at the given timestamp.
     *
     * @param to - The key of the channel to write to.
     * @param start - The starting timestamp of the first sample in data.
     * @param data  - The telemetry to write. This telemetry must have the same
     * data type as the channel.
     * @throws if the channel does not exist.
     */
    write(to: string, start: UnparsedTimeStamp, data: TypedArray): Promise<boolean>;
    /**
     * Reads telemetry from the channel between the two timestamps.
     *
     * @param from - The key of the channel to read from.
     * @param start - The starting timestamp of the range to read from.
     * @param end - The ending timestamp of the range to read from.
     * @returns a typed array containing the retrieved telemetry.
     * @throws if the channel does not exist.
     * @throws if the telemetry between start and end is not contiguous.
     */
    read(from: string, start: UnparsedTimeStamp, end: UnparsedTimeStamp): Promise<TypedArray>;
    /**
     * Reads a segment from the channel between the two timestamps.
     *
     * @param from - The key of the channel to read from.
     * @param start - The starting timestamp of the range to read from.
     * @param end - The ending timestamp of the range to read from.
     * @returns a segment containing the retrieved telemetry.
     * @throws if the channel does not exist.
     * @throws if the telemetry between start and end is not contiguous.
     */
    readSegment(from: string, start: UnparsedTimeStamp, end: UnparsedTimeStamp): Promise<TypedSegment>;
}
