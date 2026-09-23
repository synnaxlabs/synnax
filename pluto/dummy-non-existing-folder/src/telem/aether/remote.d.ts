import { channel, type framer, status as cstatus } from "@synnaxlabs/client";
import { bounds, breaker, MultiSeries, type Series, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
import { type CreateOptions } from "./factory";
import { AbstractSource, type NumberSource, type NumberSourceSpec, type SeriesSource, type SeriesSourceSpec, type Spec, type StringSource, type StringSourceSpec, type Telem } from "./telem";
/** The slice of a Synnax client that remote telemetry sources consume. */
export interface Client {
    feed: Pick<framer.Feed, "read" | "stream">;
    channels: {
        retrieve: (ch: channel.Key | channel.Name) => Promise<channel.Channel>;
    };
}
/** Reported by remote sources created while the cluster is disconnected. */
export declare const DISCONNECTED_STATUS: cstatus.Crude;
export declare const streamChannelValuePropsZ: z.ZodObject<{
    channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
}, z.core.$strip>;
export type StreamChannelValueProps = z.infer<typeof streamChannelValuePropsZ>;
export declare class StreamChannelValue extends AbstractSource<typeof streamChannelValuePropsZ> implements NumberSource {
    static readonly TYPE = "stream-channel-value";
    schema: z.ZodObject<{
        channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
    }, z.core.$strip>;
    private readonly client;
    private removeStreamHandler;
    private leadingBuffer;
    private generation;
    private valid;
    private readonly onStatusChange?;
    constructor(client: Client | null, props: unknown, options?: CreateOptions);
    /** @returns the leading series buffer for testing purposes. */
    get testingOnlyLeadingBuffer(): Series | null;
    /** @returns the internal valid flag for testing purposes */
    get testingOnlyValid(): boolean;
    cleanup(): void;
    value(): number;
    /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
    private read;
}
declare const channelDataSourcePropsZ: z.ZodObject<{
    timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        start: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<TimeSpan, TimeSpan>, z.ZodTransform<TimeStamp, TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
        end: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<TimeSpan, TimeSpan>, z.ZodTransform<TimeStamp, TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
    }, z.core.$strip>, z.ZodTransform<TimeRange, {
        start: TimeStamp;
        end: TimeStamp;
    }>>, z.ZodCustom<TimeRange, TimeRange>]>;
    channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
    useIndexOfChannel: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export type ChannelDataProps = z.input<typeof channelDataSourcePropsZ>;
export declare class ChannelData extends AbstractSource<typeof channelDataSourcePropsZ> implements SeriesSource {
    static readonly TYPE = "series-source";
    private readonly client;
    schema: z.ZodObject<{
        timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            start: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<TimeStamp, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<TimeSpan, TimeSpan>, z.ZodTransform<TimeStamp, TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
            end: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<TimeStamp, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<TimeSpan, TimeSpan>, z.ZodTransform<TimeStamp, TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
        }, z.core.$strip>, z.ZodTransform<TimeRange, {
            start: TimeStamp;
            end: TimeStamp;
        }>>, z.ZodCustom<TimeRange, TimeRange>]>;
        channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
        useIndexOfChannel: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>;
    private data;
    private valid;
    private generation;
    private channel;
    private readonly onStatusChange?;
    private readonly skipLoading;
    constructor(client: Client | null, props: unknown, options?: CreateOptions);
    cleanup(): void;
    loading(): boolean;
    value(): [bounds.Bounds, MultiSeries];
    /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
    private read;
}
declare const streamChannelDataPropsZ: z.ZodObject<{
    channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
    useIndexOfChannel: z.ZodDefault<z.ZodBoolean>;
    timeSpan: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeSpan, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<TimeStamp, TimeStamp>, z.ZodTransform<TimeSpan, TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>;
    keepFor: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeSpan, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<TimeStamp, TimeStamp>, z.ZodTransform<TimeSpan, TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
}, z.core.$strip>;
export type StreamChannelDataProps = z.input<typeof streamChannelDataPropsZ>;
export declare class StreamChannelData extends AbstractSource<typeof streamChannelDataPropsZ> implements SeriesSource {
    static readonly TYPE = "dynamic-series-source";
    private readonly client;
    private readonly data;
    private readonly now;
    private readonly onStatusChange?;
    private channel;
    private stopStreaming?;
    private valid;
    private readonly skipLoading;
    private generation;
    private readonly breaker;
    private readonly retryNotifier;
    private lastFailure?;
    schema: z.ZodObject<{
        channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
        useIndexOfChannel: z.ZodDefault<z.ZodBoolean>;
        timeSpan: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<TimeStamp, TimeStamp>, z.ZodTransform<TimeSpan, TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>;
        keepFor: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<TimeStamp, TimeStamp>, z.ZodTransform<TimeSpan, TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    }, z.core.$strip>;
    constructor(client: Client | null, props: unknown, options?: CreateOptions, now?: () => TimeStamp, breakerConfig?: breaker.Config);
    loading(): boolean;
    value(): [bounds.Bounds, MultiSeries];
    /**
     * Never rejects. A connectivity failure retries under the breaker; a definitive
     * rejection parks the source. Every distinct failure reaches onStatusChange.
     */
    private read;
    private attempt;
    private reportFailure;
    private pushNew;
    private gcOutOfRangeData;
    cleanup(): void;
}
export declare class StreamChannelStringValue extends AbstractSource<typeof streamChannelValuePropsZ> implements StringSource {
    static readonly TYPE = "stream-channel-string-value";
    schema: z.ZodObject<{
        channel: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
    }, z.core.$strip>;
    private readonly client;
    private removeStreamHandler;
    private leadingBuffer;
    private latest;
    private decodedAt;
    private generation;
    private valid;
    private readonly onStatusChange?;
    constructor(client: Client | null, props: unknown, options?: CreateOptions);
    cleanup(): void;
    value(): string;
    /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
    private read;
}
export declare class RemoteFactory {
    type: string;
    private readonly client;
    constructor(client: Client | null);
    create(spec: Spec, options?: CreateOptions): Telem | null;
}
export declare const channelData: (props: ChannelDataProps) => SeriesSourceSpec;
export declare const streamChannelData: (props: StreamChannelDataProps) => SeriesSourceSpec;
export declare const streamChannelValue: (props: Omit<StreamChannelValueProps, "units">) => NumberSourceSpec;
export declare const streamChannelStringValue: (props: StreamChannelValueProps) => StringSourceSpec;
export {};
//# sourceMappingURL=remote.d.ts.map