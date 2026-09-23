import { bounds } from "@synnaxlabs/x";
import { type AxisRenderProps, BaseAxis, baseAxisStateZ } from "./axis";
import { YAxis } from "./YAxis";
import { range } from "../range/aether";
import { type FindResult } from "../../vis/line/aether/line";
export declare const xAxisStateZ: import("zod").ZodObject<{
    tickSpacing: import("zod").ZodDefault<import("zod").ZodNumber>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
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
    type: import("zod").ZodDefault<import("zod").ZodEnum<{
        linear: "linear";
        time: "time";
    }>>;
    font: import("zod").ZodOptional<import("zod").ZodString>;
    showGrid: import("zod").ZodDefault<import("zod").ZodBoolean>;
    location: import("zod").ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>;
    gridColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
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
    axisKey: import("zod").ZodOptional<import("zod").ZodString>;
    bounds: import("zod").ZodOptional<import("zod").ZodObject<{
        lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
    }, import("zod/v4/core").$strip>>;
    manualBounds: import("zod").ZodPrefault<import("zod").ZodObject<{
        lower: import("zod").ZodDefault<import("zod").ZodBoolean>;
        upper: import("zod").ZodDefault<import("zod").ZodBoolean>;
    }, import("zod/v4/core").$strip>>;
    autoBoundPadding: import("zod").ZodOptional<import("zod").ZodNumber>;
    autoBoundUpdateInterval: import("zod").ZodDefault<import("zod").ZodUnion<readonly [import("zod").ZodPipe<import("zod").ZodObject<{
        value: import("zod").ZodBigInt;
    }, import("zod/v4/core").$strip>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, {
        value: bigint;
    }>>, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, import("zod").ZodPipe<import("zod").ZodBigInt, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, import("zod").ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, import("zod").ZodPipe<import("zod").ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, import("zod").ZodPipe<import("zod").ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    size: import("zod").ZodDefault<import("zod").ZodNumber>;
    label: import("zod").ZodDefault<import("zod").ZodString>;
    labelSize: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>;
export interface XAxisRenderProps extends AxisRenderProps {
    exposure: number;
}
export declare class XAxis extends BaseAxis<typeof baseAxisStateZ, YAxis | range.Provider> {
    static readonly TYPE = "XAxis";
    schema: import("zod").ZodObject<{
        tickSpacing: import("zod").ZodDefault<import("zod").ZodNumber>;
        color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
            rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
        }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
            r: import("zod").ZodInt;
            g: import("zod").ZodInt;
            b: import("zod").ZodInt;
            a: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
            rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
        }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
            r: import("zod").ZodInt;
            g: import("zod").ZodInt;
            b: import("zod").ZodInt;
            a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
        }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
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
        type: import("zod").ZodDefault<import("zod").ZodEnum<{
            linear: "linear";
            time: "time";
        }>>;
        font: import("zod").ZodOptional<import("zod").ZodString>;
        showGrid: import("zod").ZodDefault<import("zod").ZodBoolean>;
        location: import("zod").ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>;
        gridColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
            rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
        }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
            r: import("zod").ZodInt;
            g: import("zod").ZodInt;
            b: import("zod").ZodInt;
            a: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
            rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
        }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
            r: import("zod").ZodInt;
            g: import("zod").ZodInt;
            b: import("zod").ZodInt;
            a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
        }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
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
        axisKey: import("zod").ZodOptional<import("zod").ZodString>;
        bounds: import("zod").ZodOptional<import("zod").ZodObject<{
            lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
            upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        }, import("zod/v4/core").$strip>>;
        manualBounds: import("zod").ZodPrefault<import("zod").ZodObject<{
            lower: import("zod").ZodDefault<import("zod").ZodBoolean>;
            upper: import("zod").ZodDefault<import("zod").ZodBoolean>;
        }, import("zod/v4/core").$strip>>;
        autoBoundPadding: import("zod").ZodOptional<import("zod").ZodNumber>;
        autoBoundUpdateInterval: import("zod").ZodDefault<import("zod").ZodUnion<readonly [import("zod").ZodPipe<import("zod").ZodObject<{
            value: import("zod").ZodBigInt;
        }, import("zod/v4/core").$strip>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, {
            value: bigint;
        }>>, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, import("zod").ZodPipe<import("zod").ZodBigInt, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, import("zod").ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, import("zod").ZodPipe<import("zod").ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, import("zod").ZodPipe<import("zod").ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, import("zod").ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
        size: import("zod").ZodDefault<import("zod").ZodNumber>;
        label: import("zod").ZodDefault<import("zod").ZodString>;
        labelSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    }, import("zod/v4/core").$strip>;
    render(props: XAxisRenderProps): void;
    findByXDecimal(props: Omit<XAxisRenderProps, "canvases">, target: number): FindResult[];
    findByXValue(props: Omit<XAxisRenderProps, "canvases">, target: number): FindResult[];
    private renderYAxes;
    get yAxes(): readonly YAxis[];
    get ranges(): readonly range.Provider[];
    get loading(): boolean;
    bounds(hold: boolean): bounds.Bounds;
    private renderRanges;
    private dataBounds;
}
//# sourceMappingURL=XAxis.d.ts.map