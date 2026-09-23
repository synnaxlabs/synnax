import { box, location, type scale } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { Draw2D } from "../../../vis/draw2d";
import { render } from "../../../vis/render";
export declare const ruleStateZ: z.ZodObject<{
    position: z.ZodOptional<z.ZodNumber>;
    pixelPosition: z.ZodOptional<z.ZodNumber>;
    dragging: z.ZodBoolean;
    lineWidth: z.ZodDefault<z.ZodNumber>;
    lineDash: z.ZodDefault<z.ZodNumber>;
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
}, z.core.$strip>;
export interface RuleProps {
    location: location.Location;
    decimalToDataScale: scale.Scale;
    plot: box.Box;
    container: box.Box;
}
interface InternalState {
    renderCtx: render.Context;
    draw: Draw2D;
}
export declare class Rule extends aether.Leaf<typeof ruleStateZ, InternalState> {
    static readonly TYPE = "Rule";
    schema: z.ZodObject<{
        position: z.ZodOptional<z.ZodNumber>;
        pixelPosition: z.ZodOptional<z.ZodNumber>;
        dragging: z.ZodBoolean;
        lineWidth: z.ZodDefault<z.ZodNumber>;
        lineDash: z.ZodDefault<z.ZodNumber>;
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
    }, z.core.$strip>;
    lastUpdateRef: number | null;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(ctx: aether.Context): void;
    updatePositions({ decimalToDataScale: scale, plot }: RuleProps): number;
    render(props: RuleProps): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=aether.d.ts.map