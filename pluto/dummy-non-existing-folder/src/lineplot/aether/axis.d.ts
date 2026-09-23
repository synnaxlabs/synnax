import { bounds, box, scale, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../aether/aether";
import { axis } from "../../vis/axis";
import { type TickType } from "../../vis/axis/ticks";
import { grid } from "../../vis/grid";
import { render } from "../../vis/render";
export declare const baseAxisStateZ: z.ZodObject<{
    tickSpacing: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    type: z.ZodDefault<z.ZodEnum<{
        linear: "linear";
        time: "time";
    }>>;
    font: z.ZodOptional<z.ZodString>;
    showGrid: z.ZodDefault<z.ZodBoolean>;
    location: z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>;
    gridColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    axisKey: z.ZodOptional<z.ZodString>;
    bounds: z.ZodOptional<z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>>;
    manualBounds: z.ZodPrefault<z.ZodObject<{
        lower: z.ZodDefault<z.ZodBoolean>;
        upper: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>>;
    autoBoundPadding: z.ZodOptional<z.ZodNumber>;
    autoBoundUpdateInterval: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeSpan, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<TimeStamp, TimeStamp>, z.ZodTransform<TimeSpan, TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    size: z.ZodDefault<z.ZodNumber>;
    label: z.ZodDefault<z.ZodString>;
    labelSize: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
export type BaseAxisState = z.infer<typeof baseAxisStateZ>;
export declare const withinSizeThreshold: (prev: number, next: number) => boolean;
export declare const EMPTY_LINEAR_BOUNDS: bounds.Bounds<number>;
export declare const emptyBounds: (type: TickType) => bounds.Bounds;
export interface AxisRenderProps {
    grid: grid.Grid;
    plot: box.Box;
    viewport: box.Box;
    container: box.Box;
    canvases: render.CanvasVariant[];
    hold: boolean;
}
export declare const autoBounds: (b: bounds.Bounds[], padding: number | undefined, type: TickType) => bounds.Bounds;
interface InternalState {
    render: render.Context;
    base: axis.Axis;
    boundSnapshot?: bounds.Bounds;
    updateBounds?: (bounds: bounds.Bounds) => void;
}
export declare class BaseAxis<S extends typeof baseAxisStateZ, C extends aether.Component = aether.Component> extends aether.Composite<S, InternalState, C> {
    afterUpdate(ctx: aether.Context): void;
    afterDelete(ctx: aether.Context): void;
    renderAxis(props: AxisRenderProps, decimalToDataScale: scale.Scale): void;
    protected iBounds(hold: boolean, fetchDataBounds: () => bounds.Bounds[]): [bounds.Bounds, Error | null];
    dataToDecimalScale(hold: boolean, fetchDataBounds: () => bounds.Bounds[], viewport: box.Box): [scale.Scale, bounds.Bounds, Error | null];
}
export {};
//# sourceMappingURL=axis.d.ts.map