"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.TypedIterator = exports.CoreIterator = void 0;
const freighter_1 = require("@synnaxlabs/freighter");
const zod_1 = require("zod");
const telem_1 = require("../telem");
const payload_1 = require("./payload");
const typed_1 = __importDefault(require("./typed"));
var Command;
(function (Command) {
    Command[Command["Open"] = 0] = "Open";
    Command[Command["Next"] = 1] = "Next";
    Command[Command["Prev"] = 2] = "Prev";
    Command[Command["First"] = 3] = "First";
    Command[Command["Last"] = 4] = "Last";
    Command[Command["NextSpan"] = 5] = "NextSpan";
    Command[Command["PrevSpan"] = 6] = "PrevSpan";
    Command[Command["NextRange"] = 7] = "NextRange";
    Command[Command["Valid"] = 8] = "Valid";
    Command[Command["Error"] = 9] = "Error";
    Command[Command["SeekFirst"] = 10] = "SeekFirst";
    Command[Command["SeekLast"] = 11] = "SeekLast";
    Command[Command["SeekLT"] = 12] = "SeekLT";
    Command[Command["SeekGE"] = 13] = "SeekGE";
})(Command || (Command = {}));
var ResponseVariant;
(function (ResponseVariant) {
    ResponseVariant[ResponseVariant["None"] = 0] = "None";
    ResponseVariant[ResponseVariant["Ack"] = 1] = "Ack";
    ResponseVariant[ResponseVariant["Data"] = 2] = "Data";
})(ResponseVariant || (ResponseVariant = {}));
const RequestSchema = zod_1.z.object({
    command: zod_1.z.nativeEnum(Command),
    span: zod_1.z.number().optional(),
    range: zod_1.z.instanceof(telem_1.TimeRange).optional(),
    stamp: zod_1.z.number().optional(),
    keys: zod_1.z.string().array().optional(),
});
const ResponseSchema = zod_1.z.object({
    variant: zod_1.z.nativeEnum(ResponseVariant),
    ack: zod_1.z.boolean(),
    command: zod_1.z.nativeEnum(Command),
    error: freighter_1.ErrorPayloadSchema.optional(),
    segments: payload_1.SegmentPayloadSchema.array().nullable(),
});
/**
 * Used to iterate over a clusters telemetry in time-order. It should not be
 * instantiated directly, and should instead be instantiated via the SegmentClient.
 *
 * Using an iterator is ideal when querying/processing large ranges of data, but
 * is relatively complex and difficult to use. If you're looking to retrieve
 *  telemetry between two timestamps, see the SegmentClient.read method.
 */
class CoreIterator {
    constructor(client, aggregate = false) {
        this.aggregate = false;
        this.values = [];
        this.client = client;
        this.aggregate = aggregate;
    }
    /**
     * Opens the iterator, configuring it to iterate over the telemetry in the
     * channels with the given keys within the provided time range.
     *
     * @param tr - The time range to iterate over.
     * @param keys - The keys of the channels to iterate over.
     */
    async open(tr, keys) {
        this.stream = await this.client.stream(CoreIterator.ENDPOINT, RequestSchema, 
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        ResponseSchema);
        await this.execute({ command: Command.Open, keys, range: tr });
        this.values = [];
    }
    /**
     * Reads the next segment for each channel in the iterator.
     *
     * @returns false if the next segment can't be found for one or more channels or
     * the iterator has accumulated an error.
     */
    async next() {
        return this.execute({ command: Command.Next });
    }
    /**
     * Reads the previous segment for each channel in the iterator.
     *
     * @returns false if the next segment can't be found for one or more channels or
     * the iterator has accumulated an error.
     */
    async prev() {
        return this.execute({ command: Command.Prev });
    }
    /**
     * Seeks to the beginning of the time range and reads the first segment of each
     * channel in the iterator.
     *
     * @returns false if no segments exists in the time range for a particular channel
     * or the iterator has accumulated an error.
     */
    async first() {
        return this.execute({ command: Command.First });
    }
    /**
     * Seeks to the end of the time range and reads the last segment of each channel
     * in the iterator.
     *
     * @returns false if no segments exists in the time range for a particular channel,
     * or the iterator has accumulated an error.
     */
    async last() {
        return this.execute({ command: Command.Last });
    }
    /**
     * Reads the next time span of telemetry for each channel in the iterator.
     *
     * @returns false if a segment satisfying the request can't be found for a
     * particular channel or the iterator has accumulated an error.
     */
    async nextSpan(span) {
        return this.execute({ command: Command.NextSpan, span });
    }
    /**
     * Reads the previous time span of telemetry for each channel in the iterator.
     *
     * @returns false if a segment satisfying the request can't be found for a particular
     * channel or the iterator has accumulated an error.
     */
    async prevSpan(span) {
        return this.execute({ command: Command.PrevSpan, span });
    }
    /**
     * Seeks the iterator to the start of the time range and reads the telemetry within
     * the range for each channel.
     *
     * @returns: False if a segment satisfying the request can't be found for a particular
     * channel or the iterator has accumulated an error.
     */
    async nextRange(range) {
        return this.execute({ command: Command.NextRange, range });
    }
    /**
     * Seeks the iterator to the first segment in the time range, but does not read
     * it. Also invalidates the iterator. The iterator will not be considered valid
     * until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    async seekFirst() {
        return this.execute({ command: Command.SeekFirst });
    }
    /** Seeks the iterator to the last segment in the time range, but does not read it.
     * Also invalidates the iterator. The iterator will not be considered valid
     * until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    async seekLast() {
        return this.execute({ command: Command.SeekLast });
    }
    /**
     * Seeks the iterator to the first segment whose start is less than or equal to
     * the provided timestamp. Also invalidates the iterator. The iterator will not be
     * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    async seekLT(stamp) {
        return this.execute({ command: Command.SeekLT, stamp });
    }
    /**
     * Seeks the iterator to the first segment whose start is greater than or equal to
     * the provided timestamp. Also invalidates the iterator. The iterator will not be
     * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
     *
     * @returns false if the iterator is not pointing to a valid segment for a particular
     * channel or has accumulated an error.
     */
    async seekGE(stamp) {
        return this.execute({ command: Command.SeekGE, stamp });
    }
    /**
     * @returns true if the iterator value contains a valid segment, and fale otherwise.
     * valid most commonly returns false when the iterator is exhausted or has
     * accumulated an error.
     */
    async valid() {
        return this.execute({ command: Command.Valid });
    }
    /**
     * Closes the iterator. An iterator MUST be closed after use, and this method
     * should probably be placed in a 'finally' block. If the iterator is not closed,
     * it may leak resources.
     */
    async close() {
        if (!this.stream)
            throw new Error('iterator.open() must be called before any other method');
        this.stream.closeSend();
        const [, exc] = await this.stream.receive();
        if (!(exc instanceof freighter_1.EOF))
            throw exc;
    }
    async execute(request) {
        if (!this.stream)
            throw new Error('iterator.open() must be called before any other method');
        const err = this.stream.send(request);
        if (err)
            throw err;
        if (!this.aggregate)
            this.values = [];
        for (;;) {
            const [res, err] = await this.stream.receive();
            if (err || !res)
                throw err;
            if (res.variant == ResponseVariant.Ack)
                return res.ack;
            if (res.segments)
                this.values.push(...res.segments);
        }
    }
}
exports.CoreIterator = CoreIterator;
CoreIterator.ENDPOINT = '/segment/iterate';
class TypedIterator extends CoreIterator {
    constructor(client, channels, aggregate = false) {
        super(client, aggregate);
        this.channels = channels;
    }
    async value() {
        const result = {};
        this.values.sort((a, b) => a.start.valueOf() - b.start.valueOf());
        const keys = this.values.map((v) => v.channelKey);
        const channels = await this.channels.getN(...keys);
        this.values.forEach((v) => {
            const sugared = new typed_1.default(channels.find((c) => c.key == v.channelKey), v);
            if (v.channelKey in result) {
                result[v.channelKey].extend(sugared);
            }
            else {
                result[v.channelKey] = sugared;
            }
        });
        return result;
    }
}
exports.TypedIterator = TypedIterator;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaXRlcmF0b3IuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3NlZ21lbnQvaXRlcmF0b3IudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7O0FBQUEscURBSytCO0FBQy9CLDZCQUF3QjtBQUl4QixvQ0FBcUM7QUFFckMsdUNBQWlFO0FBQ2pFLG9EQUFtQztBQUVuQyxJQUFLLE9BZUo7QUFmRCxXQUFLLE9BQU87SUFDVixxQ0FBUSxDQUFBO0lBQ1IscUNBQVEsQ0FBQTtJQUNSLHFDQUFRLENBQUE7SUFDUix1Q0FBUyxDQUFBO0lBQ1QscUNBQVEsQ0FBQTtJQUNSLDZDQUFZLENBQUE7SUFDWiw2Q0FBWSxDQUFBO0lBQ1osK0NBQWEsQ0FBQTtJQUNiLHVDQUFTLENBQUE7SUFDVCx1Q0FBUyxDQUFBO0lBQ1QsZ0RBQWMsQ0FBQTtJQUNkLDhDQUFhLENBQUE7SUFDYiwwQ0FBVyxDQUFBO0lBQ1gsMENBQVcsQ0FBQTtBQUNiLENBQUMsRUFmSSxPQUFPLEtBQVAsT0FBTyxRQWVYO0FBRUQsSUFBSyxlQUlKO0FBSkQsV0FBSyxlQUFlO0lBQ2xCLHFEQUFRLENBQUE7SUFDUixtREFBTyxDQUFBO0lBQ1AscURBQVEsQ0FBQTtBQUNWLENBQUMsRUFKSSxlQUFlLEtBQWYsZUFBZSxRQUluQjtBQUVELE1BQU0sYUFBYSxHQUFHLE9BQUMsQ0FBQyxNQUFNLENBQUM7SUFDN0IsT0FBTyxFQUFFLE9BQUMsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDO0lBQzlCLElBQUksRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsUUFBUSxFQUFFO0lBQzNCLEtBQUssRUFBRSxPQUFDLENBQUMsVUFBVSxDQUFDLGlCQUFTLENBQUMsQ0FBQyxRQUFRLEVBQUU7SUFDekMsS0FBSyxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxRQUFRLEVBQUU7SUFDNUIsSUFBSSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxLQUFLLEVB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