import { channel } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
import { type LogEntry, type LogSource, type LogSourceSpec } from "./types";
import { type CreateOptions } from "../../../telem/aether/factory";
import { type Client } from "../../../telem/aether/remote";
import { AbstractSource } from "../../../telem/aether/telem";
declare const streamMultiChannelLogPropsZ: z.ZodObject<{
    channels: z.ZodArray<z.ZodUnion<[z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>, z.ZodString]>>;
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
export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;
export declare class StreamMultiChannelLog extends AbstractSource<typeof streamMultiChannelLogPropsZ> implements LogSource {
    static readonly TYPE = "stream-multi-channel-log";
    schema: z.ZodObject<{
        channels: z.ZodArray<z.ZodUnion<[z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>, z.ZodString]>>;
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
    private readonly client;
    private readonly onStatusChange?;
    private readonly now;
    private readonly maxEntries;
    private channelMeta;
    private entries;
    private stopStreaming?;
    private valid;
    private _evictedCount;
    private _channels;
    private readGeneration;
    get evictedCount(): number;
    constructor(client: Client | null, props: unknown, options?: CreateOptions, now?: () => TimeStamp, maxEntries?: number);
    value(): LogEntry[];
    setChannels(channels: channel.Key[]): void;
    /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
    private read;
    private gcEntries;
    cleanup(): void;
}
export declare const streamMultiChannelLog: (props: StreamMultiChannelLogProps) => LogSourceSpec;
export {};
//# sourceMappingURL=sources.d.ts.map