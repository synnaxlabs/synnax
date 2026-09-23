import { z } from "zod";
/**
 * configZ carries the staleness keys a symbol config holds. Extend a symbol's schema
 * with its shape to pair the schema with {@link Fields}, which writes these exact keys.
 * Both are optional, because a symbol saved before it gained staleness config carries
 * neither.
 */
export declare const configZ: z.ZodObject<{
    stalenessTimeout: z.ZodOptional<z.ZodNumber>;
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
}, z.core.$strip>;
export interface Config extends z.infer<typeof configZ> {
}
/** The staleness a symbol starts with. An unset color resolves to the theme warning. */
export declare const ZERO_CONFIG: Config;
//# sourceMappingURL=config.d.ts.map