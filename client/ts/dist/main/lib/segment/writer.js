"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.TypedWriter = exports.CoreWriter = void 0;
const freighter_1 = require("@synnaxlabs/freighter");
const zod_1 = require("zod");
const errors_1 = require("../errors");
const telem_1 = require("../telem");
const payload_1 = require("./payload");
const splitter_1 = __importDefault(require("./splitter"));
const typed_1 = __importDefault(require("./typed"));
const validator_1 = require("./validator");
const RequestSchema = zod_1.z.object({
    openKeys: zod_1.z.string().array().optional(),
    segments: payload_1.SegmentPayloadSchema.array().optional(),
});
const ResponseSchema = zod_1.z.object({
    ack: zod_1.z.boolean(),
    error: freighter_1.ErrorPayloadSchema.optional(),
});
const NOT_OPEN = new errors_1.GeneralError('Writer has not been opened. Please open before calling write() or close().');
/**
 * CoreWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
class CoreWriter {
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
        if (!(res === null || res === void 0 ? void 0 : res.ack))
            throw new errors_1.UnexpectedError('Writer failed to positively acknowledge open request. This is a bug. Please report it.');
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
        if (!err && (res === null || res === void 0 ? void 0 : res.error))
            throw (0, freighter_1.decodeError)(res.error);
        if (!(err instanceof freighter_1.EOF))
            throw err;
    }
    checkKeys(segments) {
        // check that the channel key of each segment is in the open keys
        segments
            .map((segment) => segment.channelKey)
            .forEach((key) => {
            if (!this.keys.includes(key))
                throw new errors_1.ValidationError({
                    field: 'channelKey',
                    message: `Channel key ${key} is not in the list of keys this writer was opened with.`,
                });
        });
    }
}
exports.CoreWriter = CoreWriter;
CoreWriter.ENDPOINT = '/segment/write';
/**
 * TypedWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
class TypedWriter {
    constructor(client, channels) {
        this.core = new CoreWriter(client);
        this.channels = channels;
        this.scalarTypeValidator = new validator_1.ScalarTypeValidator();
        this.contiguityValidator = new validator_1.ContiguityValidator({
            allowNoHighWaterMark: true,
            allowGaps: false,
            allowOverlap: false,
        });
        this.splitter = new splitter_1.default(telem_1.Size.Megabytes(4));
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
            start: new telem_1.TimeStamp(start),
            data: new Uint8Array(data.buffer),
        };
        const segment = new typed_1.default(ch, pld);
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
exports.TypedWriter = TypedWriter;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid3JpdGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L3dyaXRlci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7Ozs7QUFBQSxxREFNK0I7QUFDL0IsNkJBQXdCO0FBR3hCLHNDQUEyRTtBQUMzRSxvQ0FBMEU7QUFFMUUsdUNBQWlFO0FBQ2pFLDBEQUFrQztBQUNsQyxvREFBbUM7QUFDbkMsMkNBQXVFO0FBRXZFLE1BQU0sYUFBYSxHQUFHLE9BQUMsQ0FBQyxNQUFNLENBQUM7SUFDN0IsUUFBUSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxLQUFLLEVBQUUsQ0FBQyxRQUFRLEVBQUU7SUFDdkMsUUFBUSxFQUFFLDhCQUFvQixDQUFDLEtBQUssRUFBRSxDQUFDLFFBQVEsRUFBRTtDQUNsRCxDQUFDLENBQUM7QUFJSCxNQUFNLGNBQWMsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQzlCLEdBQUcsRUFBRSxPQUFDLENBQUMsT0FBTyxFQUFFO0lBQ2hCLEtBQUssRUFBRSw4QkFBa0IsQ0FBQyxRQUFRLEVBQUU7Q0FDckMsQ0FBQyxDQUFDO0FBSUgsTUFBTSxRQUFRLEdBQUcsSUFBSSxxQkFBWSxDQUMvQiw0RUFBNEUsQ0FDN0UsQ0FBQztBQUVGOzs7Ozs7OztHQVFHO0FBQ0gsTUFBYSxVQUFVO0lBTXJCLFlBQVksTUFBb0I7UUFDOUIsSUFBSSxDQUFDLE1BQU0sR0FBRyxNQUFNLENBQUM7UUFDckIsSUFBSSxDQUFDLElBQUksR0FBRyxFQUFFLENBQUM7SUFDakIsQ0FBQztJQUVEOzs7Ozs7O09BT0c7SUFDSCxLQUFLLENBQUMsSUFBSSxDQUFDLElBQWM7UUFDdkIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLENBQUM7UUFDakIsSUFBSSxDQUFDLE1BQU0sR0FBRyxNQUFNLElBQUksQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUNwQyxVQUFVLENBQUMsUUFBUTtRQUNuQiw2REFBNkQ7UUFD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