import { bounds, scale } from "@synnaxlabs/x";
import { type AxisRenderProps, BaseAxis, baseAxisStateZ } from "./axis";
import { rule } from "../rule/aether";
import { line } from "../../vis/line/aether";
export declare const yAxisStateZ: import("zod").ZodObject<{
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
    location: import("zod").ZodDefault<import("zod").ZodEnum<{
        left: "left";
        right: "right";
    }>>;
}, import("zod/v4/core").$strip>;
export interface YAxisProps extends AxisRenderProps {
    xDataToDecimalScale: scale.Scale;
    /** Bounds of the parent x axis; y bounds cover only samples inside them. */
    xBounds: bounds.Bounds;
    exposure: number;
}
type Children = line.Line | rule.Rule;
export declare class YAxis extends BaseAxis<typeof baseAxisStateZ, Children> {
    static readonly TYPE = "YAxis";
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
    xBounds(): bounds.Bounds;
    bounds(hold: boolean, xBounds: bounds.Bounds): bounds.Bounds;
    render(props: YAxisProps): void;
    private renderLines;
    private renderRules;
    findByXValue({ xDataToDecimalScale, xBounds, plot, viewport, hold, exposure, }: Omit<YAxisProps, "canvases">, target: number): line.FindResult[];
    get loading(): boolean;
    private dataBounds;
    private get lines();
    private get rules();
}
export {};
//# sourceMappingURL=YAxis.d.ts.map