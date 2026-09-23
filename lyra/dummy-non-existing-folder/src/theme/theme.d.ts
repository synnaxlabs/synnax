import { z } from "zod";
export type Shade = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11;
export declare const themeZ: z.ZodPipe<z.ZodObject<{
    name: z.ZodString;
    key: z.ZodString;
    colors: z.ZodObject<{
        border: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        primary: z.ZodUnion<[z.ZodObject<{
            m2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            m1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            z: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>, z.ZodPipe<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        } | [number, number, number] | [number, number, number, number]>>, z.ZodTransform<{
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        }, [number, number, number, number]>>]>;
        gray: z.ZodObject<{
            l0: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l3: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l4: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l5: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l6: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l7: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l8: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l9: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l10: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            l11: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>;
        error: z.ZodUnion<[z.ZodObject<{
            m2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            m1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            z: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>, z.ZodPipe<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        } | [number, number, number] | [number, number, number, number]>>, z.ZodTransform<{
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        }, [number, number, number, number]>>]>;
        secondary: z.ZodUnion<[z.ZodObject<{
            m2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            m1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            z: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>, z.ZodPipe<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        } | [number, number, number] | [number, number, number, number]>>, z.ZodTransform<{
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        }, [number, number, number, number]>>]>;
        warning: z.ZodUnion<[z.ZodObject<{
            m2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            m1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            z: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p1: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            p2: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>, z.ZodPipe<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        } | [number, number, number] | [number, number, number, number]>>, z.ZodTransform<{
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        }, [number, number, number, number]>>]>;
        palettes: z.ZodRecord<z.ZodString, z.ZodObject<{
            key: z.ZodString;
            name: z.ZodString;
            swatches: z.ZodArray<z.ZodObject<{
                key: z.ZodString;
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
            }, z.core.$strip>>;
        }, z.core.$strip>>;
        visualization: z.ZodDefault<z.ZodObject<{
            palettes: z.ZodRecord<z.ZodString, z.ZodArray<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        }, z.core.$strip>>;
        white: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        black: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        text: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        textInverted: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        textOnPrimary: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        textOnWarning: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        primaryText: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        errorText: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        warningText: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
        logo: z.ZodString;
    }, z.core.$strip>;
    sizes: z.ZodObject<{
        base: z.ZodNumber;
        border: z.ZodObject<{
            radius: z.ZodObject<{
                tiny: z.ZodNumber;
                small: z.ZodNumber;
                medium: z.ZodNumber;
                large: z.ZodNumber;
                huge: z.ZodNumber;
            }, z.core.$strip>;
            thickWidth: z.ZodNumber;
        }, z.core.$strip>;
        schematic: z.ZodObject<{
            elementStrokeWidth: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>;
    typography: z.ZodObject<{
        family: z.ZodString;
        codeFamily: z.ZodString;
        h1: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        h2: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        h3: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        h4: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        h5: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        p: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
        small: z.ZodObject<{
            size: z.ZodNumber;
            weight: z.ZodUnion<readonly [z.ZodNumber, z.ZodEnum<{
                bold: "bold";
                bolder: "bolder";
                lighter: "lighter";
                normal: "normal";
            }>]>;
            lineHeight: z.ZodNumber;
            textTransform: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>;
    }, z.core.$strip>;
}, z.core.$strip>, z.ZodTransform<{
    name: string;
    key: string;
    colors: {
        border: [number, number, number, number];
        primary: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        gray: {
            l0: [number, number, number, number];
            l1: [number, number, number, number];
            l2: [number, number, number, number];
            l3: [number, number, number, number];
            l4: [number, number, number, number];
            l5: [number, number, number, number];
            l6: [number, number, number, number];
            l7: [number, number, number, number];
            l8: [number, number, number, number];
            l9: [number, number, number, number];
            l10: [number, number, number, number];
            l11: [number, number, number, number];
        };
        error: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        secondary: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        warning: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        palettes: Record<string, {
            key: string;
            name: string;
            swatches: {
                key: string;
                name: string;
                color: [number, number, number, number];
            }[];
        }>;
        visualization: {
            palettes: Record<string, [number, number, number, number][]>;
        };
        white: [number, number, number, number];
        black: [number, number, number, number];
        text: [number, number, number, number];
        textInverted: [number, number, number, number];
        textOnPrimary: [number, number, number, number];
        textOnWarning: [number, number, number, number];
        primaryText: [number, number, number, number];
        errorText: [number, number, number, number];
        warningText: [number, number, number, number];
        logo: string;
    };
    sizes: {
        base: number;
        border: {
            radius: {
                tiny: number;
                small: number;
                medium: number;
                large: number;
                huge: number;
            };
            thickWidth: number;
        };
        schematic: {
            elementStrokeWidth: number;
        };
    };
    typography: {
        family: string;
        codeFamily: string;
        h1: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h2: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h3: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h4: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h5: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        p: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        small: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
    };
}, {
    name: string;
    key: string;
    colors: {
        border: [number, number, number, number];
        primary: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        gray: {
            l0: [number, number, number, number];
            l1: [number, number, number, number];
            l2: [number, number, number, number];
            l3: [number, number, number, number];
            l4: [number, number, number, number];
            l5: [number, number, number, number];
            l6: [number, number, number, number];
            l7: [number, number, number, number];
            l8: [number, number, number, number];
            l9: [number, number, number, number];
            l10: [number, number, number, number];
            l11: [number, number, number, number];
        };
        error: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        secondary: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        warning: {
            m2: [number, number, number, number];
            m1: [number, number, number, number];
            z: [number, number, number, number];
            p1: [number, number, number, number];
            p2: [number, number, number, number];
        } | {
            readonly m2: [number, number, number, number];
            readonly m1: [number, number, number, number];
            readonly z: [number, number, number, number];
            readonly p1: [number, number, number, number];
            readonly p2: [number, number, number, number];
        };
        palettes: Record<string, {
            key: string;
            name: string;
            swatches: {
                key: string;
                name: string;
                color: [number, number, number, number];
            }[];
        }>;
        visualization: {
            palettes: Record<string, [number, number, number, number][]>;
        };
        white: [number, number, number, number];
        black: [number, number, number, number];
        text: [number, number, number, number];
        textInverted: [number, number, number, number];
        textOnPrimary: [number, number, number, number];
        textOnWarning: [number, number, number, number];
        primaryText: [number, number, number, number];
        errorText: [number, number, number, number];
        warningText: [number, number, number, number];
        logo: string;
    };
    sizes: {
        base: number;
        border: {
            radius: {
                tiny: number;
                small: number;
                medium: number;
                large: number;
                huge: number;
            };
            thickWidth: number;
        };
        schematic: {
            elementStrokeWidth: number;
        };
    };
    typography: {
        family: string;
        codeFamily: string;
        h1: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h2: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h3: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h4: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        h5: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        p: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
        small: {
            size: number;
            weight: number | "bold" | "bolder" | "lighter" | "normal";
            lineHeight: number;
            textTransform?: string | undefined;
        };
    };
}>>;
export type ThemeSpec = z.input<typeof themeZ>;
export type Theme = z.infer<typeof themeZ>;
export declare const SYNNAX_LIGHT: ThemeSpec;
export declare const SYNNAX_DARK: ThemeSpec;
export declare const SYNNAX_THEMES: {
    synnaxDark: {
        name: string;
        key: string;
        colors: {
            border: string | {
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
            } | [number, number, number] | [number, number, number, number];
            primary: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            gray: {
                l0: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l3: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l4: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l5: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l6: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l7: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l8: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l9: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l10: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l11: string | {
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
                } | [number, number, number] | [number, number, number, number];
            };
            error: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            secondary: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            warning: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            palettes: Record<string, {
                key: string;
                name: string;
                swatches: {
                    key: string;
                    name: string;
                    color: string | {
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
                    } | [number, number, number] | [number, number, number, number];
                }[];
            }>;
            visualization?: {
                palettes: Record<string, (string | {
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
                } | [number, number, number] | [number, number, number, number])[]>;
            } | undefined;
            white: string | {
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
            } | [number, number, number] | [number, number, number, number];
            black: string | {
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
            } | [number, number, number] | [number, number, number, number];
            text: string | {
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
            } | [number, number, number] | [number, number, number, number];
            textInverted: string | {
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
            } | [number, number, number] | [number, number, number, number];
            textOnPrimary?: string | {
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
            } | [number, number, number] | [number, number, number, number] | undefined;
            textOnWarning?: string | {
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
            } | [number, number, number] | [number, number, number, number] | undefined;
            primaryText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            errorText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            warningText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            logo: string;
        };
        sizes: {
            base: number;
            border: {
                radius: {
                    tiny: number;
                    small: number;
                    medium: number;
                    large: number;
                    huge: number;
                };
                thickWidth: number;
            };
            schematic: {
                elementStrokeWidth: number;
            };
        };
        typography: {
            family: string;
            codeFamily: string;
            h1: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h2: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h3: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h4: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h5: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            p: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            small: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
        };
    };
    synnaxLight: {
        name: string;
        key: string;
        colors: {
            border: string | {
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
            } | [number, number, number] | [number, number, number, number];
            primary: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            gray: {
                l0: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l3: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l4: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l5: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l6: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l7: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l8: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l9: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l10: string | {
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
                } | [number, number, number] | [number, number, number, number];
                l11: string | {
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
                } | [number, number, number] | [number, number, number, number];
            };
            error: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            secondary: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            warning: string | {
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
            } | {
                m2: string | {
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
                } | [number, number, number] | [number, number, number, number];
                m1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                z: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p1: string | {
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
                } | [number, number, number] | [number, number, number, number];
                p2: string | {
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
                } | [number, number, number] | [number, number, number, number];
            } | [number, number, number] | [number, number, number, number];
            palettes: Record<string, {
                key: string;
                name: string;
                swatches: {
                    key: string;
                    name: string;
                    color: string | {
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
                    } | [number, number, number] | [number, number, number, number];
                }[];
            }>;
            visualization?: {
                palettes: Record<string, (string | {
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
                } | [number, number, number] | [number, number, number, number])[]>;
            } | undefined;
            white: string | {
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
            } | [number, number, number] | [number, number, number, number];
            black: string | {
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
            } | [number, number, number] | [number, number, number, number];
            text: string | {
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
            } | [number, number, number] | [number, number, number, number];
            textInverted: string | {
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
            } | [number, number, number] | [number, number, number, number];
            textOnPrimary?: string | {
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
            } | [number, number, number] | [number, number, number, number] | undefined;
            textOnWarning?: string | {
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
            } | [number, number, number] | [number, number, number, number] | undefined;
            primaryText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            errorText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            warningText: string | {
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
            } | [number, number, number] | [number, number, number, number];
            logo: string;
        };
        sizes: {
            base: number;
            border: {
                radius: {
                    tiny: number;
                    small: number;
                    medium: number;
                    large: number;
                    huge: number;
                };
                thickWidth: number;
            };
            schematic: {
                elementStrokeWidth: number;
            };
        };
        typography: {
            family: string;
            codeFamily: string;
            h1: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h2: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h3: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h4: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            h5: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            p: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
            small: {
                size: number;
                weight: number | "bold" | "bolder" | "lighter" | "normal";
                lineHeight: number;
                textTransform?: string | undefined;
            };
        };
    };
};
//# sourceMappingURL=theme.d.ts.map