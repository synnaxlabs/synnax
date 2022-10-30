import { decodeError, EOF, ErrorPayloadSchema, } from '@synnaxlabs/freighter';
import { z } from 'zod';
import { GeneralError, UnexpectedError, ValidationError } from '../errors';
import { Size, TimeStamp } from '../telem';
import { SegmentPayloadSchema } from './payload';
import Splitter from './splitter';
import TypedSegment from './typed';
import { ContiguityValidator, ScalarTypeValidator } from './validator';
const RequestSchema = z.object({
    openKeys: z.string().array().optional(),
    segments: SegmentPayloadSchema.array().optional(),
});
const ResponseSchema = z.object({
    ack: z.boolean(),
    error: ErrorPayloadSchema.optional(),
});
const NOT_OPEN = new GeneralError('Writer has not been opened. Please open before calling write() or close().');
/**
 * CoreWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
export class CoreWriter {
    static ENDPOINT = '/segment/write';
    client;
    stream;
    keys;
    constructor(client) {
        this.client = client;
        this.keys = [];
    }
    /**
     * Opens the writer, acquiring an exclusive lock on the given channels for
     * the duration of the writer's lifetime. open must be called before any other
     * writer methods.
     *
     * @param keys - A list of keys representing the channels the writer will write
     * to.
     */
    async open(keys) {
        this.keys = keys;
        this.stream = await this.client.stream(CoreWriter.ENDPOINT, 
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        RequestSchema, ResponseSchema);
        this.stream.send({ openKeys: keys });
        const [res, err] = await this.stream.receive();
        if (err)
            throw err;
        if (!res?.ack)
            throw new UnexpectedError('Writer failed to positively acknowledge open request. This is a bug. Please report it.');
    }
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
    async write(segments) {
        if (!this.stream)
            throw NOT_OPEN;
        if (this.stream.received())
            return false;
        this.checkKeys(segments);
        const err = this.stream.send({ segments });
        if (err)
            throw err;
        return true;
    }
    /**
     * Closes the writer, raising any accumulated error encountered during operation.
     * A writer MUST be closed after use, and this method should probably be placed
     * in a 'finally' block. If the writer is not closed, the database will not release
     * the exclusive lock on the channels, preventing any other callers from
     * writing to them. It also might leak resources and threads.
     */
    async close() {
        if (!this.stream)
            throw NOT_OPEN;
        this.stream.closeSend();
        const [res, err] = await this.stream.receive();
        if (!err && res?.error)
            throw decodeError(res.error);
        if (!(err instanceof EOF))
            throw err;
    }
    checkKeys(segments) {
        // check that the channel key of each segment is in the open keys
        segments
            .map((segment) => segment.channelKey)
            .forEach((key) => {
            if (!this.keys.includes(key))
                throw new ValidationError({
                    field: 'channelKey',
                    message: `Channel key ${key} is not in the list of keys this writer was opened with.`,
                });
        });
    }
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
export class TypedWriter {
    core;
    splitter;
    channels;
    scalarTypeValidator;
    contiguityValidator;
    constructor(client, channels) {
        this.core = new CoreWriter(client);
        this.channels = channels;
        this.scalarTypeValidator = new ScalarTypeValidator();
        this.contiguityValidator = new ContiguityValidator({
            allowNoHighWaterMark: true,
            allowGaps: false,
            allowOverlap: false,
        });
        this.splitter = new Splitter(Size.Megabytes(4));
    }
    /**
     * Opens the writer, acquiring an exclusive lock on the given channels for
     * the duration of the writer's lifetime. open must be called before any other
     * writer methods.
     *
     * @param keys - A list of keys representing the channels the writer will write
     * to.
     */
    async open(keys) {
        await this.core.open(keys);
    }
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
    async write(to, start, data) {
        const ch = await this.channels.get(to);
        this.scalarTypeValidator.validate(data, ch.dataType);
        const pld = {
            channelKey: to,
            start: new TimeStamp(start),
            data: new Uint8Array(data.buffer),
        };
        const segment = new TypedSegment(ch, pld);
        this.contiguityValidator.validate(segment);
        const segments = this.splitter.split(segment);
        return this.core.write(segments.map((s) => s.payload));
    }
    /**
     * Closes the writer, raising any accumulated error encountered during operation.
     * A writer MUST be closed after use, and this method should probably be placed
     * in a 'finally' block. If the writer is not closed, the database will not release
     * the exclusive lock on the channels, preventing any other callers from
     * writing to them. It also might leak resources and threads.
     */
    async close() {
        await this.core.close();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid3JpdGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L3dyaXRlci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQ0wsV0FBVyxFQUNYLEdBQUcsRUFDSCxrQkFBa0IsR0FHbkIsTUFBTSx1QkFBdUIsQ0FBQztBQUMvQixPQUFPLEVBQUUsQ0FBQyxFQUFFLE1BQU0sS0FBSyxDQUFDO0FBR3hCLE9BQU8sRUFBRSxZQUFZLEVBQUUsZUFBZSxFQUFFLGVBQWUsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUMzRSxPQUFPLEVBQUUsSUFBSSxFQUFFLFNBQVMsRUFBaUMsTUFBTSxVQUFVLENBQUM7QUFFMUUsT0FBTyxFQUFrQixvQkFBb0IsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUNqRSxPQUFPLFFBQVEsTUFBTSxZQUFZLENBQUM7QUFDbEMsT0FBTyxZQUFZLE1BQU0sU0FBUyxDQUFDO0FBQ25DLE9BQU8sRUFBRSxtQkFBbUIsRUFBRSxtQkFBbUIsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUV2RSxNQUFNLGFBQWEsR0FBRyxDQUFDLENBQUMsTUFBTSxDQUFDO0lBQzdCLFFBQVEsRUFBRSxDQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsS0FBSyxFQUFFLENBQUMsUUFBUSxFQUFFO0lBQ3ZDLFFBQVEsRUFBRSxvQkFBb0IsQ0FBQyxLQUFLLEVBQUUsQ0FBQyxRQUFRLEVBQUU7Q0FDbEQsQ0FBQyxDQUFDO0FBSUgsTUFBTSxjQUFjLEdBQUcsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUM5QixHQUFHLEVBQUUsQ0FBQyxDQUFDLE9BQU8sRUFBRTtJQUNoQixLQUFLLEVBQUUsa0JBQWtCLENBQUMsUUFBUSxFQUFFO0NBQ3JDLENBQUMsQ0FBQztBQUlILE1BQU0sUUFBUSxHQUFHLElBQUksWUFBWSxDQUMvQiw0RUFBNEUsQ0FDN0UsQ0FBQztBQUVGOzs7Ozs7OztHQVFHO0FBQ0gsTUFBTSxPQUFPLFVBQVU7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