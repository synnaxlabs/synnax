import { type z } from "zod";
export declare const basePropsZ: z.ZodObject<{
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
    backgroundTelem: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"color">;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    } | [number, number, number] | [number, number, number, number]>>>>;
    precision: z.ZodDefault<z.ZodNumber>;
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
    useWidthForBackground: z.ZodDefault<z.ZodBoolean>;
    valueBackgroundShift: z.ZodDefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    valueBackgroundOverScan: z.ZodDefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    clip: z.ZodDefault<z.ZodBoolean>;
    borderRadius: z.ZodOptional<z.ZodUnion<readonly [z.ZodNumber, z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>, z.ZodObject<{
        topLeft: z.ZodNumber;
        topRight: z.ZodNumber;
        bottomLeft: z.ZodNumber;
        bottomRight: z.ZodNumber;
    }, z.core.$strip>, z.ZodObject<{
        topLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        topRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>]>>;
    level: z.ZodOptional<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
}, z.core.$strip>;
export interface UseProps extends z.input<typeof basePropsZ> {
    aetherKey: string;
}
export interface UseReturn {
    width: number;
}
export declare const use: ({ aetherKey, box, telem, color, precision, minWidth, level, backgroundTelem, notation, location, stalenessColor, stalenessTimeout, useWidthForBackground, valueBackgroundOverScan, valueBackgroundShift, clip, borderRadius, }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map