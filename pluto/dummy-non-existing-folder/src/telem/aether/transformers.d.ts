import { status } from "@synnaxlabs/client";
import { color, type math, MultiSeries } from "@synnaxlabs/x";
import { z } from "zod";
import { type Factory } from "./factory";
import { type BooleanSink, type BooleanSinkSpec, type BooleanSource, type BooleanSourceSpec, type ColorSourceSpec, MultiSourceTransformer, type NumberSourceSpec, type SeriesSourceSpec, type Spec, type StringSourceSpec, type Telem, UnarySinkTransformer, UnarySourceTransformer } from "./telem";
export declare class TransformerFactory implements Factory {
    type: string;
    create(spec: Spec): Telem | null;
}
declare const setpointProps: z.ZodObject<{
    truthy: z.ZodDefault<z.ZodNumber>;
    falsy: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
export type SetpointProps = z.infer<typeof setpointProps>;
export declare const setpoint: (props: SetpointProps) => BooleanSinkSpec;
export declare class SetPoint extends UnarySinkTransformer<boolean, number, typeof setpointProps> implements BooleanSink {
    static readonly TYPE = "boolean-numeric-converter-sink";
    static readonly propsZ: z.ZodObject<{
        truthy: z.ZodDefault<z.ZodNumber>;
        falsy: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        truthy: z.ZodDefault<z.ZodNumber>;
        falsy: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    transform(...values: boolean[]): number[];
}
export declare const withinBoundsProps: z.ZodObject<{
    trueBound: z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>;
}, z.core.$strip>;
export type WithinBoundsProps = z.infer<typeof withinBoundsProps>;
export declare const withinBounds: (props: WithinBoundsProps) => BooleanSourceSpec;
export declare class WithinBounds extends UnarySourceTransformer<number, boolean, typeof withinBoundsProps> implements BooleanSource {
    static readonly TYPE = "boolean-source";
    static readonly propsZ: z.ZodObject<{
        trueBound: z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        trueBound: z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>;
    }, z.core.$strip>;
    protected transform(value: number): boolean;
}
declare const meanProps: z.ZodObject<{}, z.core.$strip>;
export declare class Mean extends MultiSourceTransformer<number, number, typeof meanProps> {
    static readonly TYPE = "mean";
    static readonly propsZ: z.ZodObject<{}, z.core.$strip>;
    schema: z.ZodObject<{}, z.core.$strip>;
    protected transform(values: Record<string, number>): number;
}
export declare const mean: (props: z.input<typeof meanProps>) => BooleanSourceSpec;
export declare const booleanStatusProps: z.ZodObject<{
    trueVariant: z.ZodDefault<z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>;
}, z.core.$strip>;
export declare class BooleanStatus extends UnarySourceTransformer<status.Status, boolean, typeof booleanStatusProps> {
    static readonly TYPE = "boolean-status";
    static readonly propsZ: z.ZodObject<{
        trueVariant: z.ZodDefault<z.ZodEnum<{
            disabled: "disabled";
            error: "error";
            info: "info";
            loading: "loading";
            success: "success";
            warning: "warning";
        }>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        trueVariant: z.ZodDefault<z.ZodEnum<{
            disabled: "disabled";
            error: "error";
            info: "info";
            loading: "loading";
            success: "success";
            warning: "warning";
        }>>;
    }, z.core.$strip>;
    protected transform(value: status.Status): boolean;
}
export declare const booleanStatus: (props: z.input<typeof booleanStatusProps>) => BooleanSourceSpec;
export declare const stringifyNumberProps: z.ZodObject<{
    precision: z.ZodDefault<z.ZodNumber>;
    prefix: z.ZodDefault<z.ZodString>;
    suffix: z.ZodDefault<z.ZodString>;
    notation: z.ZodDefault<z.ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
}, z.core.$strip>;
export declare class StringifyNumber extends UnarySourceTransformer<math.Numeric, string, typeof stringifyNumberProps> {
    static readonly TYPE = "stringify-number";
    static readonly propsZ: z.ZodObject<{
        precision: z.ZodDefault<z.ZodNumber>;
        prefix: z.ZodDefault<z.ZodString>;
        suffix: z.ZodDefault<z.ZodString>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        precision: z.ZodDefault<z.ZodNumber>;
        prefix: z.ZodDefault<z.ZodString>;
        suffix: z.ZodDefault<z.ZodString>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
    }, z.core.$strip>;
    protected transform(value: math.Numeric): string;
}
export declare const stringifyNumber: (props: z.input<typeof stringifyNumberProps>) => StringSourceSpec;
export declare const rollingAverageProps: z.ZodObject<{
    windowSize: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
export declare class RollingAverage extends UnarySourceTransformer<math.Numeric, number, typeof rollingAverageProps> {
    static readonly TYPE = "rolling-average";
    static readonly propsZ: z.ZodObject<{
        windowSize: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        windowSize: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    private readonly window;
    protected transform(value: math.Numeric): number;
    protected shouldNotify(value: math.Numeric): boolean;
}
export declare const rollingAverage: (props: z.input<typeof rollingAverageProps>) => NumberSourceSpec;
export declare const colorGradientProps: z.ZodObject<{
    gradient: z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        color: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>]>;
        position: z.ZodNumber;
        switched: z.ZodOptional<z.ZodBoolean>;
    }, z.core.$strip>>;
}, z.core.$strip>;
export declare class ColorGradient extends UnarySourceTransformer<number, color.Color, typeof colorGradientProps> {
    static readonly TYPE = "color-gradient";
    static readonly propsZ: z.ZodObject<{
        gradient: z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            color: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            }, z.core.$strip>]>;
            position: z.ZodNumber;
            switched: z.ZodOptional<z.ZodBoolean>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        gradient: z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            color: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            }, z.core.$strip>]>;
            position: z.ZodNumber;
            switched: z.ZodOptional<z.ZodBoolean>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    protected transform(value: number): color.Color;
}
export declare const colorGradient: (props: z.input<typeof colorGradientProps>) => ColorSourceSpec;
export declare const scaleNumberProps: z.ZodObject<{
    scale: z.ZodObject<{
        offset: z.ZodNumber;
        scale: z.ZodNumber;
    }, z.core.$strip>;
}, z.core.$strip>;
export declare class ScaleNumber extends UnarySourceTransformer<math.Numeric, number, typeof scaleNumberProps> {
    static readonly TYPE = "scale-number";
    static readonly propsZ: z.ZodObject<{
        scale: z.ZodObject<{
            offset: z.ZodNumber;
            scale: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        scale: z.ZodObject<{
            offset: z.ZodNumber;
            scale: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>;
    protected transform(value: math.Numeric): number;
}
export declare const scaleNumber: (props: z.input<typeof scaleNumberProps>) => NumberSourceSpec;
export declare const downsampleModeZ: z.ZodEnum<{
    average: "average";
    decimate: "decimate";
}>;
export type DownsampleMode = z.infer<typeof downsampleModeZ>;
export declare const downsampleModeProps: z.ZodObject<{
    mode: z.ZodEnum<{
        average: "average";
        decimate: "decimate";
    }>;
    windowSize: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
export type DownsampleModeProps = z.infer<typeof downsampleModeProps>;
export declare const downsampleMode: (props: DownsampleModeProps) => NumberSourceSpec;
export declare class SeriesDownsampler {
    static readonly TYPE = "series-downsampler";
    private _downsample;
    private readonly cache;
    readonly props: DownsampleModeProps;
    constructor(props: DownsampleModeProps);
    private downsample;
    transform(source: MultiSeries): MultiSeries;
}
export declare const seriesDownsampler: (props: z.input<typeof downsampleModeProps>) => SeriesSourceSpec;
export {};
//# sourceMappingURL=transformers.d.ts.map