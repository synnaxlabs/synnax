import { bounds, color, MultiSeries, Rate, Series } from "@synnaxlabs/x";
import { z } from "zod";
import { type Factory } from "./factory";
import { AbstractSource, type ColorSource, type ColorSourceSpec, type NumberSource, type NumberSourceSpec, type SeriesSource, type SeriesSourceSpec, type Spec, type StringSourceSpec, type Telem } from "./telem";
export declare class StaticFactory implements Factory {
    type: string;
    create(spec: Spec): Telem | null;
}
export declare const fixedSeriesPropsZ: z.ZodObject<{
    data: z.ZodArray<z.ZodUnion<readonly [z.ZodCustom<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>, z.ZodCustom<Uint16Array<ArrayBuffer>, Uint16Array<ArrayBuffer>>, z.ZodCustom<Uint32Array<ArrayBuffer>, Uint32Array<ArrayBuffer>>, z.ZodCustom<BigUint64Array<ArrayBuffer>, BigUint64Array<ArrayBuffer>>, z.ZodCustom<Float32Array<ArrayBuffer>, Float32Array<ArrayBuffer>>, z.ZodCustom<Float64Array<ArrayBuffer>, Float64Array<ArrayBuffer>>, z.ZodCustom<Int8Array<ArrayBuffer>, Int8Array<ArrayBuffer>>, z.ZodCustom<Int16Array<ArrayBuffer>, Int16Array<ArrayBuffer>>, z.ZodCustom<Int32Array<ArrayBuffer>, Int32Array<ArrayBuffer>>, z.ZodCustom<BigInt64Array<ArrayBuffer>, BigInt64Array<ArrayBuffer>>]>>;
    offsets: z.ZodDefault<z.ZodArray<z.ZodNumber>>;
}, z.core.$strip>;
export type FixedArrayProps = z.input<typeof fixedSeriesPropsZ>;
export declare const iterativeSeriesPropsZ: z.ZodObject<{
    data: z.ZodArray<z.ZodUnion<readonly [z.ZodCustom<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>, z.ZodCustom<Uint16Array<ArrayBuffer>, Uint16Array<ArrayBuffer>>, z.ZodCustom<Uint32Array<ArrayBuffer>, Uint32Array<ArrayBuffer>>, z.ZodCustom<BigUint64Array<ArrayBuffer>, BigUint64Array<ArrayBuffer>>, z.ZodCustom<Float32Array<ArrayBuffer>, Float32Array<ArrayBuffer>>, z.ZodCustom<Float64Array<ArrayBuffer>, Float64Array<ArrayBuffer>>, z.ZodCustom<Int8Array<ArrayBuffer>, Int8Array<ArrayBuffer>>, z.ZodCustom<Int16Array<ArrayBuffer>, Int16Array<ArrayBuffer>>, z.ZodCustom<Int32Array<ArrayBuffer>, Int32Array<ArrayBuffer>>, z.ZodCustom<BigInt64Array<ArrayBuffer>, BigInt64Array<ArrayBuffer>>]>>;
    offsets: z.ZodDefault<z.ZodArray<z.ZodNumber>>;
    rate: z.ZodUnion<readonly [z.ZodPipe<z.ZodNumber, z.ZodTransform<Rate, number>>, z.ZodCustom<Rate, Rate>]>;
    yOffset: z.ZodDefault<z.ZodNumber>;
    scroll: z.ZodDefault<z.ZodNumber>;
    startPosition: z.ZodDefault<z.ZodNumber>;
    scrollBounds: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export type IterativeArrayProps = z.input<typeof iterativeSeriesPropsZ>;
export declare class IterativeSeries extends AbstractSource<typeof iterativeSeriesPropsZ> implements SeriesSource {
    static readonly TYPE = "iterative-series";
    schema: z.ZodObject<{
        data: z.ZodArray<z.ZodUnion<readonly [z.ZodCustom<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>, z.ZodCustom<Uint16Array<ArrayBuffer>, Uint16Array<ArrayBuffer>>, z.ZodCustom<Uint32Array<ArrayBuffer>, Uint32Array<ArrayBuffer>>, z.ZodCustom<BigUint64Array<ArrayBuffer>, BigUint64Array<ArrayBuffer>>, z.ZodCustom<Float32Array<ArrayBuffer>, Float32Array<ArrayBuffer>>, z.ZodCustom<Float64Array<ArrayBuffer>, Float64Array<ArrayBuffer>>, z.ZodCustom<Int8Array<ArrayBuffer>, Int8Array<ArrayBuffer>>, z.ZodCustom<Int16Array<ArrayBuffer>, Int16Array<ArrayBuffer>>, z.ZodCustom<Int32Array<ArrayBuffer>, Int32Array<ArrayBuffer>>, z.ZodCustom<BigInt64Array<ArrayBuffer>, BigInt64Array<ArrayBuffer>>]>>;
        offsets: z.ZodDefault<z.ZodArray<z.ZodNumber>>;
        rate: z.ZodUnion<readonly [z.ZodPipe<z.ZodNumber, z.ZodTransform<Rate, number>>, z.ZodCustom<Rate, Rate>]>;
        yOffset: z.ZodDefault<z.ZodNumber>;
        scroll: z.ZodDefault<z.ZodNumber>;
        startPosition: z.ZodDefault<z.ZodNumber>;
        scrollBounds: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>;
    position: number;
    interval?: number;
    data: Series[];
    constructor(props: unknown);
    value(): [bounds.Bounds, MultiSeries];
    start(rate: Rate): void;
    cleanup(): void;
}
export declare const fixedNumberPropsZ: z.ZodNumber;
export type FixedNumberProps = z.infer<typeof fixedNumberPropsZ>;
export declare class FixedNumber extends AbstractSource<typeof fixedNumberPropsZ> implements NumberSource {
    static readonly TYPE = "static-numeric";
    schema: z.ZodNumber;
    value(): number;
}
export declare const fixedStringPropsZ: z.ZodString;
export type FixedStringProps = z.infer<typeof fixedStringPropsZ>;
export declare class FixedString extends AbstractSource<typeof fixedStringPropsZ> {
    static readonly TYPE = "static-string";
    schema: z.ZodString;
    value(): string;
}
export declare const fixedColorSourcePropsZ: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
export type FixedColorSourceProps = z.infer<typeof fixedColorSourcePropsZ>;
export declare class FixedColorSource extends AbstractSource<typeof fixedColorSourcePropsZ> implements ColorSource {
    static readonly TYPE = "static-color";
    schema: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    value(): color.Color;
}
export declare const fixedArray: (props: FixedArrayProps) => SeriesSourceSpec;
export declare const iterativeArray: (props: IterativeArrayProps) => SeriesSourceSpec;
export declare const fixedNumber: (value: number) => NumberSourceSpec;
export declare const fixedString: (value: string) => StringSourceSpec;
export declare const fixedColor: (color: color.Crude) => ColorSourceSpec;
//# sourceMappingURL=static.d.ts.map