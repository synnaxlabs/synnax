import { TimeRange } from '../telem';
import { TypedIterator } from './iterator';
import { TypedWriter } from './writer';
export default class SegmentClient {
    transport;
    channels;
    constructor(transport, channels) {
        this.transport = transport;
        this.channels = channels;
    }
    /**
     * Opens a new iterator over the given channels within the provided time range.
     *
     * @param tr - A time range to iterate over.
     * @param keys - A list of channel keys to iterate over.
     * @param aggregate - Whether to accumulate iteration results or reset them
     * on every iterator method call.
     * @returns a new {@link TypedIterator}.
     */
    async newIterator(tr, keys, aggregate) {
        const iter = new TypedIterator(this.transport.streamClient, this.channels, aggregate);
        await iter.open(tr, keys);
        return iter;
    }
    /**
     * Opens a new writer on the given channels.
     *
     * @param keys - The keys of the channels to write to. A writer cannot write to
     * a channel that is not in this list. See the {@link TypedWriter} documentation
     * for more information.
     * @returns a new {@link TypedWriter}.
     */
    async newWriter(keys) {
        const writer = new TypedWriter(this.transport.streamClient, this.channels);
        await writer.open(keys);
        return writer;
    }
    /**
     * Writes telemetry to the given channel starting at the given timestamp.
     *
     * @param to - The key of the channel to write to.
     * @param start - The starting timestamp of the first sample in data.
     * @param data  - The telemetry to write. This telemetry must have the same
     * data type as the channel.
     * @throws if the channel does not exist.
     */
    async write(to, start, data) {
        const writer = await this.newWriter([to]);
        try {
            return await writer.write(to, start, data);
        }
        finally {
            await writer.close();
        }
    }
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
    async read(from, start, end) {
        return (await this.readSegment(from, start, end)).view;
    }
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
    async readSegment(from, start, end) {
        const iter = await this.newIterator(new TimeRange(start, end), [from], true);
        let seg;
        try {
            await iter.first();
            // eslint-disable-next-line no-empty
            while (await iter.next()) { }
            seg = (await iter.value())[from];
        }
        finally {
            await iter.close();
        }
        return seg;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L2NsaWVudC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsU0FBUyxFQUFpQyxNQUFNLFVBQVUsQ0FBQztBQUdwRSxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBRTNDLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFFdkMsTUFBTSxDQUFDLE9BQU8sT0FBTyxhQUFhO0lBQ3hCLFNBQVMsQ0FBWTtJQUNyQixRQUFRLENBQVc7SUFFM0IsWUFBWSxTQUFvQixFQUFFLFFBQWtCO1FBQ2xELElBQUksQ0FBQyxTQUFTLEdBQUcsU0FBUyxDQUFDO1FBQzNCLElBQUksQ0FBQyxRQUFRLEdBQUcsUUFBUSxDQUFDO0lBQzNCLENBQUM7SUFFRDs7Ozs7Ozs7T0FRRztJQUNILEtBQUssQ0FBQyxXQUFXLENBQ2YsRUFBYSxFQUNiLElBQWMsRUFDZCxTQUFrQjtRQUVsQixNQUFNLElBQUksR0FBRyxJQUFJLGFBQWEsQ0FDNUIsSUFBSSxDQUFDLFNBQVMsQ0FBQyxZQUFZLEVBQzNCLElBQUksQ0FBQyxRQUFRLEVBQ2IsU0FBUyxDQUNWLENBQUM7UUFDRixNQUFNLElBQUksQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLElBQUksQ0FBQyxDQUFDO1FBQzFCLE9BQU8sSUFBSSxDQUFDO0lBQ2QsQ0FBQztJQUVEOzs7Ozs7O09BT0c7SUFDSCxLQUFLLENBQUMsU0FBUyxDQUFDLElBQWM7UUFDNUIsTUFBTSxNQUFNLEdBQUcsSUFBSSxXQUFXLENBQUMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxZQUFZLEVBQUUsSUFBSSxDQUFDLFFBQVEsQ0FBQyxDQUFDO1FBQzNFLE1BQU0sTUFBTSxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztRQUN4QixPQUFPLE1BQU0sQ0FBQztJQUNoQixDQUFDO0lBRUQ7Ozs7Ozs7O09BUUc7SUFDSCxLQUFLLENBQUMsS0FBSyxDQUNULEVBQVUsRUFDVixLQUF3QixFQUN4QixJQUFnQjtRQUVoQixNQUFNLE1BQU0sR0FBRyxNQUFNLElBQUksQ0FBQyxTQUFTLENBQUMsQ0FBQyxFQUFFLENBQUMsQ0FBQyxDQUFDO1FBQzFDLElBQUk7WUFDRixPQUFPLE1BQU0sTUFBTSxDQUFDLEtBQUssQ0FBQyxFQUFFLEVBQUUsS0FBSyxFQUFFLElBQUksQ0FBQyxDQUFDO1NBQzVDO2dCQUFTO1lBQ1IsTUFBTSxNQUFNLENBQUMsS0FBSyxFQUFFLENBQUM7U0FDdEI7SUFDSCxDQUFDO0lBRUQ7Ozs7Ozs7OztPQVNHO0lBQ0gsS0FBSyxDQUFDLElBQUksQ0FDUixJQUFZLEVBQ1osS0FBd0IsRUFDeEIsR0FBc0I7UUFFdEIsT0FBTyxDQUFDLE1BQU0sSUFBSSxDQUFDLFdBQVcsQ0FBQyxJQUFJLEVBQUUsS0FBSyxFQUFFLEdBQUcsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO0lBQ3pELENBQUM7SUFFRDs7Ozs7Ozs7O09BU0c7SUFDSCxLQUFLLENBQUMsV0FBVyxDQUNmLElBQVksRUFDWixLQUF3QixFQUN4QixHQUFzQjtRQUV0QixNQUFNLElBQUksR0FBRyxNQUFNLElBQUksQ0FBQyxXQUFXLENBQ2pDLElBQUksU0FBUyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsRUFDekIsQ0FBQyxJQUFJLENBQUMsRUFDTixJQUFJLENBQ0wsQ0FBQztRQUNGLElBQUksR0FBaUIsQ0FBQztRQUN0QixJQUFJO1lBQ0YsTUFBTSxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7WUFDbkIsb0NBQW9DO1lBQ3BDLE9BQU8sTUFBTSxJQUFJLENBQUMsSUFBSSxFQUFFLEVBQUUsR0FBRTtZQUM1QixHQUFHLEdBQUcsQ0FBQyxNQUFNLElBQUksQ0FBQyxLQUFLLEVBQUUsQ0FBQyxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ2xDO2dCQUFTO1lBQ1IsTUFBTSxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7U0FDcEI7UUFDRCxPQUFPLEdBQW1CLENBQUM7SUFDN0IsQ0FBQztDQUNGIn0=