import { TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { Draw2D } from "../../../vis/draw2d";
import { render } from "../../../vis/render";
export declare const annotationStateZ: z.ZodObject<{
    start: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeStamp, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
    end: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeStamp, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
}, z.core.$strip>;
interface InternalState {
    render: render.Context;
    draw: Draw2D;
}
export declare class Annotation extends aether.Leaf<typeof annotationStateZ, InternalState> {
    static readonly TYPE = "range-annotation";
    schema: z.ZodObject<{
        start: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
        end: z.ZodUnion<readonly [z.ZodCustom<TimeStamp, TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
}
export {};
//# sourceMappingURL=annotation.d.ts.map