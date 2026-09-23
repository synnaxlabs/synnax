import { ranger, type Synnax } from "@synnaxlabs/client";
import { box, type scale, TimeRange } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { flux } from "../../../flux/aether";
import { ranger as aetherRanger } from "../../../ranger/aether";
import { status } from "../../../status/aether";
import { Draw2D } from "../../../vis/draw2d";
import { render } from "../../../vis/render";
export declare const selectedStateZ: z.ZodObject<{
    key: z.ZodUUID;
    name: z.ZodString;
    timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        start: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
        end: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
    }, z.core.$strip>, z.ZodTransform<TimeRange, {
        start: import("@synnaxlabs/x").TimeStamp;
        end: import("@synnaxlabs/x").TimeStamp;
    }>>, z.ZodCustom<TimeRange, TimeRange>]>;
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
    labels: z.ZodOptional<z.ZodPipe<z.ZodNullable<z.ZodArray<z.ZodObject<{
        key: z.ZodDefault<z.ZodUUID>;
        name: z.ZodString;
        color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        } | [number, number, number] | [number, number, number, number]>>;
    }, z.core.$strip>>>, z.ZodTransform<{
        key: string;
        name: string;
        color: [number, number, number, number];
    }[] | undefined, {
        key: string;
        name: string;
        color: [number, number, number, number];
    }[] | null>>>;
    parent: z.ZodOptional<typeof ranger.payloadZ>;
    viewport: z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>;
}, z.core.$strip>;
export type SelectedState = z.infer<typeof selectedStateZ>;
export declare const providerStateZ: z.ZodObject<{
    cursor: z.ZodUnion<[z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>, z.ZodNull]>;
    visible: z.ZodDefault<z.ZodOptional<z.ZodBoolean>>;
    hovered: z.ZodUnion<[z.ZodObject<{
        key: z.ZodUUID;
        name: z.ZodString;
        timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            start: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
            end: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
        }, z.core.$strip>, z.ZodTransform<TimeRange, {
            start: import("@synnaxlabs/x").TimeStamp;
            end: import("@synnaxlabs/x").TimeStamp;
        }>>, z.ZodCustom<TimeRange, TimeRange>]>;
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
        labels: z.ZodOptional<z.ZodPipe<z.ZodNullable<z.ZodArray<z.ZodObject<{
            key: z.ZodDefault<z.ZodUUID>;
            name: z.ZodString;
            color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            } | [number, number, number] | [number, number, number, number]>>;
        }, z.core.$strip>>>, z.ZodTransform<{
            key: string;
            name: string;
            color: [number, number, number, number];
        }[] | undefined, {
            key: string;
            name: string;
            color: [number, number, number, number];
        }[] | null>>>;
        parent: z.ZodOptional<typeof ranger.payloadZ>;
        viewport: z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>;
    }, z.core.$strip>, z.ZodNull]>;
    count: z.ZodNumber;
}, z.core.$strip>;
export type ProviderState = z.infer<typeof providerStateZ>;
interface InternalState {
    retrieve: flux.Retrieve<aetherRanger.ListQuery, ranger.Range[]>;
    client: Synnax | null;
    render: render.Context;
    requestRender: render.Requestor;
    draw: Draw2D;
    runAsync: status.ErrorHandler;
}
export interface ProviderProps {
    dataToDecimalScale: scale.Scale;
    viewport: box.Box;
    region: box.Box;
    timeRange: TimeRange;
}
/**
 * The most ranges the annotation strip draws for one window. A denser window draws
 * none: the strip cannot label them legibly in 32 pixels, and the Core answers a
 * limited query in key order, so drawing a truncated answer would show an arbitrary
 * subset that changes as the window moves.
 */
export declare const MAX_ANNOTATIONS = 100;
export declare class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
    static readonly TYPE = "range-provider";
    static readonly stateZ: z.ZodObject<{
        cursor: z.ZodUnion<[z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>, z.ZodNull]>;
        visible: z.ZodDefault<z.ZodOptional<z.ZodBoolean>>;
        hovered: z.ZodUnion<[z.ZodObject<{
            key: z.ZodUUID;
            name: z.ZodString;
            timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                start: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
                end: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
            }, z.core.$strip>, z.ZodTransform<TimeRange, {
                start: import("@synnaxlabs/x").TimeStamp;
                end: import("@synnaxlabs/x").TimeStamp;
            }>>, z.ZodCustom<TimeRange, TimeRange>]>;
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
            labels: z.ZodOptional<z.ZodPipe<z.ZodNullable<z.ZodArray<z.ZodObject<{
                key: z.ZodDefault<z.ZodUUID>;
                name: z.ZodString;
                color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
                } | [number, number, number] | [number, number, number, number]>>;
            }, z.core.$strip>>>, z.ZodTransform<{
                key: string;
                name: string;
                color: [number, number, number, number];
            }[] | undefined, {
                key: string;
                name: string;
                color: [number, number, number, number];
            }[] | null>>>;
            parent: z.ZodOptional<typeof ranger.payloadZ>;
            viewport: z.ZodObject<{
                lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
                upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            }, z.core.$strip>;
        }, z.core.$strip>, z.ZodNull]>;
        count: z.ZodNumber;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        cursor: z.ZodUnion<[z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>, z.ZodNull]>;
        visible: z.ZodDefault<z.ZodOptional<z.ZodBoolean>>;
        hovered: z.ZodUnion<[z.ZodObject<{
            key: z.ZodUUID;
            name: z.ZodString;
            timeRange: z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                start: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
                end: z.ZodUnion<readonly [z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, bigint>>, z.ZodPipe<z.ZodDate, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, Date>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeSpan>>, z.ZodPipe<z.ZodUnion<readonly [z.ZodTuple<[z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>]>, z.ZodTransform<import("@synnaxlabs/x").TimeStamp, [number] | [number, number] | [number, number, number]>>]>;
            }, z.core.$strip>, z.ZodTransform<TimeRange, {
                start: import("@synnaxlabs/x").TimeStamp;
                end: import("@synnaxlabs/x").TimeStamp;
            }>>, z.ZodCustom<TimeRange, TimeRange>]>;
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
            labels: z.ZodOptional<z.ZodPipe<z.ZodNullable<z.ZodArray<z.ZodObject<{
                key: z.ZodDefault<z.ZodUUID>;
                name: z.ZodString;
                color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
                } | [number, number, number] | [number, number, number, number]>>;
            }, z.core.$strip>>>, z.ZodTransform<{
                key: string;
                name: string;
                color: [number, number, number, number];
            }[] | undefined, {
                key: string;
                name: string;
                color: [number, number, number, number];
            }[] | null>>>;
            parent: z.ZodOptional<typeof ranger.payloadZ>;
            viewport: z.ZodObject<{
                lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
                upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            }, z.core.$strip>;
        }, z.core.$strip>, z.ZodNull]>;
        count: z.ZodNumber;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    render(props: ProviderProps): void;
}
export {};
//# sourceMappingURL=provider.d.ts.map