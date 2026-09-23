import { type theme } from "@synnaxlabs/lyra/theme";
import { color, scale, text, xy } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
import { type Element } from "../../diagram/aether/Diagram";
import { Draw2D } from "../../draw2d";
import { render } from "../../render";
import { staleness } from "../../staleness/aether";
export declare const GAUGE_SIZES: {
    readonly small: 80;
    readonly medium: 120;
    readonly large: 160;
    readonly huge: 200;
};
export type GaugeSize = keyof typeof GAUGE_SIZES;
export declare const gaugeSizeZ: z.ZodEnum<{
    huge: "huge";
    large: "large";
    medium: "medium";
    small: "small";
}>;
declare const gaugeState: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    box: z.ZodObject<{
        one: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        two: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        root: z.ZodObject<{
            x: z.ZodEnum<{
                left: "left";
                right: "right";
            }>;
            y: z.ZodEnum<{
                bottom: "bottom";
                top: "top";
            }>;
        }, z.core.$strip>;
    }, z.core.$strip>;
    telem: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"string">;
    }, z.core.$strip>>;
    level: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    precision: z.ZodDefault<z.ZodNumber>;
    minWidth: z.ZodDefault<z.ZodNumber>;
    width: z.ZodOptional<z.ZodNumber>;
    notation: z.ZodDefault<z.ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
    location: z.ZodDefault<z.ZodObject<{
        x: z.ZodUnion<[z.ZodEnum<{
            left: "left";
            right: "right";
        }>, z.ZodEnum<{
            center: "center";
        }>]>;
        y: z.ZodUnion<[z.ZodEnum<{
            bottom: "bottom";
            top: "top";
        }>, z.ZodEnum<{
            center: "center";
        }>]>;
    }, z.core.$strip>>;
    units: z.ZodDefault<z.ZodString>;
    bounds: z.ZodDefault<z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>>;
    barWidth: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
export interface GaugeProps {
    scale?: scale.XY;
}
interface InternalState {
    theme: theme.Theme;
    render: render.Context;
    telem: telem.StringSource;
    draw2d: Draw2D;
    stopListening?: () => void;
    staleness: staleness.Registration;
    stale: boolean;
    requestRender: render.Requestor | null;
    textColor: color.Color;
    strokeColor: color.Color;
    gaugeStartAngle: number;
    gaugeEndAngle: number;
    gaugeAngleRange: number;
    outerRadius: number;
    innerRadius: number;
    labelRadius: number;
    labelInwardShift: number;
    minLabelPos: xy.XY;
    maxLabelPos: xy.XY;
    centerPos: xy.XY;
    valueTextPos: xy.XY;
    unitsTextPos: xy.XY;
    textLevel: text.Level;
    labelLevel: text.Level;
}
export declare class Gauge extends aether.Leaf<typeof gaugeState, InternalState> implements Element {
    static readonly TYPE = "gauge";
    static readonly z: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        box: z.ZodObject<{
            one: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            two: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            root: z.ZodObject<{
                x: z.ZodEnum<{
                    left: "left";
                    right: "right";
                }>;
                y: z.ZodEnum<{
                    bottom: "bottom";
                    top: "top";
                }>;
            }, z.core.$strip>;
        }, z.core.$strip>;
        telem: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"string">;
        }, z.core.$strip>>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        precision: z.ZodDefault<z.ZodNumber>;
        minWidth: z.ZodDefault<z.ZodNumber>;
        width: z.ZodOptional<z.ZodNumber>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        location: z.ZodDefault<z.ZodObject<{
            x: z.ZodUnion<[z.ZodEnum<{
                left: "left";
                right: "right";
            }>, z.ZodEnum<{
                center: "center";
            }>]>;
            y: z.ZodUnion<[z.ZodEnum<{
                bottom: "bottom";
                top: "top";
            }>, z.ZodEnum<{
                center: "center";
            }>]>;
        }, z.core.$strip>>;
        units: z.ZodDefault<z.ZodString>;
        bounds: z.ZodDefault<z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>>;
        barWidth: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        box: z.ZodObject<{
            one: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            two: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            root: z.ZodObject<{
                x: z.ZodEnum<{
                    left: "left";
                    right: "right";
                }>;
                y: z.ZodEnum<{
                    bottom: "bottom";
                    top: "top";
                }>;
            }, z.core.$strip>;
        }, z.core.$strip>;
        telem: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"string">;
        }, z.core.$strip>>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        precision: z.ZodDefault<z.ZodNumber>;
        minWidth: z.ZodDefault<z.ZodNumber>;
        width: z.ZodOptional<z.ZodNumber>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        location: z.ZodDefault<z.ZodObject<{
            x: z.ZodUnion<[z.ZodEnum<{
                left: "left";
                right: "right";
            }>, z.ZodEnum<{
                center: "center";
            }>]>;
            y: z.ZodUnion<[z.ZodEnum<{
                bottom: "bottom";
                top: "top";
            }>, z.ZodEnum<{
                center: "center";
            }>]>;
        }, z.core.$strip>>;
        units: z.ZodDefault<z.ZodString>;
        bounds: z.ZodDefault<z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>>;
        barWidth: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    private requestRender;
    render({ viewportScale }: {
        viewportScale?: scale.XY | undefined;
    }): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=gauge.d.ts.map