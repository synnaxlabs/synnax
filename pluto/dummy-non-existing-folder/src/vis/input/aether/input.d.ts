import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
import { type diagram } from "../../diagram/aether";
export declare const stateZ: z.ZodObject<{
    sink: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"sink">;
        valueType: z.ZodLiteral<"string">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export interface State extends z.input<typeof stateZ> {
}
export declare const methodsZ: {
    set: z.ZodFunction<z.ZodTuple<[z.ZodString], null>, z.ZodVoid>;
};
interface InternalState {
    sink: telem.StringSink;
}
export declare class Input extends aether.Leaf<typeof stateZ, InternalState, typeof methodsZ> implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ> {
    static readonly TYPE = "Input";
    static readonly METHODS: {
        set: z.ZodFunction<z.ZodTuple<[z.ZodString], null>, z.ZodVoid>;
    };
    schema: z.ZodObject<{
        sink: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
            valueType: z.ZodLiteral<"string">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    methods: {
        set: z.ZodFunction<z.ZodTuple<[z.ZodString], null>, z.ZodVoid>;
    };
    afterUpdate(ctx: aether.Context): void;
    set(value: string): void;
    afterDelete(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=input.d.ts.map