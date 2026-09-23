import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
export declare const MODES: readonly ["fire", "momentary", "pulse"];
export declare const modeZ: z.ZodEnum<{
    fire: "fire";
    momentary: "momentary";
    pulse: "pulse";
}>;
export type Mode = z.infer<typeof modeZ>;
export declare const buttonStateZ: z.ZodObject<{
    sink: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"sink">;
        valueType: z.ZodLiteral<"boolean">;
    }, z.core.$strip>>;
    mode: z.ZodDefault<z.ZodEnum<{
        fire: "fire";
        momentary: "momentary";
        pulse: "pulse";
    }>>;
}, z.core.$strip>;
export declare const buttonMethodsZ: {
    onMouseDown: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    onMouseUp: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
};
interface InternalState {
    sink: telem.BooleanSink;
}
export declare class Button extends aether.Leaf<typeof buttonStateZ, InternalState, typeof buttonMethodsZ> implements aether.HandlersFromSchema<typeof buttonMethodsZ> {
    static readonly TYPE = "Button";
    static readonly METHODS: {
        onMouseDown: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
        onMouseUp: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    static readonly z: z.ZodObject<{
        sink: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
        mode: z.ZodDefault<z.ZodEnum<{
            fire: "fire";
            momentary: "momentary";
            pulse: "pulse";
        }>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        sink: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
        mode: z.ZodDefault<z.ZodEnum<{
            fire: "fire";
            momentary: "momentary";
            pulse: "pulse";
        }>>;
    }, z.core.$strip>;
    methods: {
        onMouseDown: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
        onMouseUp: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    afterUpdate(ctx: aether.Context): void;
    onMouseDown(): void;
    onMouseUp(): void;
    afterDelete(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=button.d.ts.map