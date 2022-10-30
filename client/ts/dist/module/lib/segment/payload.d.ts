import { z } from 'zod';
import { TimeStamp } from '../telem';
export declare const SegmentPayloadSchema: z.ZodObject<{
    channelKey: z.ZodString;
    start: z.ZodEffects<z.ZodNumber, TimeStamp, number>;
    data: z.ZodEffects<z.ZodString, Uint8Array, string>;
}, "strip", z.ZodTypeAny, {
    data: Uint8Array;
    channelKey: string;
    start: TimeStamp;
}, {
    data: string;
    channelKey: string;
    start: number;
}>;
export declare type SegmentPayload = z.infer<typeof SegmentPayloadSchema>;
