import { ChannelPayload } from '../channel/payload';
import { Size, TimeRange, TimeSpan, TimeStamp, TypedArray } from '../telem';
import { SegmentPayload } from './payload';
export default class TypedSegment {
    payload: SegmentPayload;
    channel: ChannelPayload;
    view: TypedArray;
    constructor(channel: ChannelPayload, payload: SegmentPayload);
    get start(): TimeStamp;
    get span(): TimeSpan;
    get range(): TimeRange;
    get end(): TimeStamp;
    get size(): Size;
    extend(other: TypedSegment): void;
}
