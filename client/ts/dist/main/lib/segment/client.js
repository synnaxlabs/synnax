"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const telem_1 = require("../telem");
const iterator_1 = require("./iterator");
const writer_1 = require("./writer");
class SegmentClient {
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
        const iter = new iterator_1.TypedIterator(this.transport.streamClient, this.channels, aggregate);
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
        const writer = new writer_1.TypedWriter(this.transport.streamClient, this.channels);
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
        const iter = await this.newIterator(new telem_1.TimeRange(start, end), [from], true);
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
exports.default = SegmentClient;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L2NsaWVudC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOztBQUNBLG9DQUFvRTtBQUdwRSx5Q0FBMkM7QUFFM0MscUNBQXVDO0FBRXZDLE1BQXFCLGFBQWE7SUFJaEMsWUFBWSxTQUFvQixFQUFFLFFBQWtCO1FBQ2xELElBQUksQ0FBQyxTQUFTLEdBQUcsU0FBUyxDQUFDO1FBQzNCLElBQUksQ0FBQyxRQUFRLEdBQUcsUUFBUSxDQUFDO0lBQzNCLENBQUM7SUFFRDs7Ozs7Ozs7T0FRRztJQUNILEtBQUssQ0FBQyxXQUFXLENBQ2YsRUFBYSxFQUNiLElBQWMsRUFDZCxTQUFrQjtRQUVsQixNQUFNLElBQUksR0FBRyxJQUFJLHdCQUFhLENBQzVCLElBQUksQ0FBQyxTQUFTLENBQUMsWUFBWSxFQUMzQixJQUFJLENBQUMsUUFBUSxFQUNiLFNBQVMsQ0FDVixDQUFDO1FBQ0YsTUFBTSxJQUFJLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxJQUFJLENBQUMsQ0FBQztRQUMxQixPQUFPLElBQUksQ0FBQztJQUNkLENBQUM7SUFFRDs7Ozs7OztPQU9HO0lBQ0gsS0FBSyxDQUFDLFNBQVMsQ0FBQyxJQUFjO1FBQzVCLE1BQU0sTUFBTSxHQUFHLElBQUksb0JBQVcsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLFlBQVksRUFBRSxJQUFJLENBQUMsUUFBUSxDQUFDLENBQUM7UUFDM0UsTUFBTSxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQ3hCLE9BQU8sTUFBTSxDQUFDO0lBQ2hCLENBQUM7SUFFRDs7Ozs7Ozs7T0FRRztJQUNILEtBQUssQ0FBQyxLQUFLLENBQ1QsRUFBVSxFQUNWLEtBQXdCLEVBQ3hCLElBQWdCO1FBRWhCLE1BQU0sTUFBTSxHQUFHLE1BQU0sSUFBSSxDQUFDLFNBQVMsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUM7UUFDMUMsSUFBSTtZQUNGLE9BQU8sTUFBTSxNQUFNLENBQUMsS0FBSyxDQUFDLEVBQUUsRUFBRSxLQUFLLEVBQUUsSUFBSSxDQUFDLENBQUM7U0FDNUM7Z0JBQVM7WUFDUixNQUFNLE1BQU0sQ0FBQyxLQUFLLEVBQUUsQ0FBQztTQUN0QjtJQUNILENBQUM7SUFFRDs7Ozs7Ozs7O09BU0c7SUFDSCxLQUFLLENBQUMsSUFBSSxDQUNSLElBQVksRUFDWixLQUF3QixFQUN4QixHQUFzQjtRQUV0QixPQUFPLENBQUMsTUFBTSxJQUFJLENBQUMsV0FBVyxDQUFDLElBQUksRUFBRSxLQUFLLEVBQUUsR0FBRyxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUM7SUFDekQsQ0FBQztJQUVEOzs7Ozs7Ozs7T0FTRztJQUNILEtBQUssQ0FBQyxXQUFXLENBQ2YsSUFBWSxFQUNaLEtBQXdCLEVBQ3hCLEdBQXNCO1FBRXRCLE1BQU0sSUFBSSxHQUFHLE1BQU0sSUFBSSxDQUFDLFdBQVcsQ0FDakMsSUFBSSxpQkFBUyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsRUFDekIsQ0FBQyxJQUFJLENBQUMsRUFDTixJQUFJLENBQ0wsQ0FBQztRQUNGLElBQUksR0FBaUIsQ0FBQztRQUN0QixJQUFJO1lBQ0YsTUFBTSxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7WUFDbkIsb0NBQW9DO1lBQ3BDLE9BQU8sTUFBTSxJQUFJLENBQUMsSUFBSSxFQUFFLEVBQUUsR0FBRTtZQUM1QixHQUFHLEdBQUcsQ0FBQyxNQUFNLElBQUksQ0FBQyxLQUFLLEVBQUUsQ0FBQyxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ2xDO2dCQUFTO1lBQ1IsTUFBTSxJQUFJLENBQUMsS0FBSyxFQUFFLENBQUM7U0FDcEI7UUFDRCxPQUFPLEdBQW1CLENBQUM7SUFDN0IsQ0FBQztDQUNGO0FBckhELGdDQXFIQyJ9