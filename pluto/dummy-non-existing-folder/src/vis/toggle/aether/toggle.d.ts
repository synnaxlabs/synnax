import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { status } from "../../../status/aether";
import { telem } from "../../../telem/aether";
import { type diagram } from "../../diagram/aether";
import { staleness } from "../../staleness/aether";
export declare const toggleStateZ: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stale: z.ZodDefault<z.ZodBoolean>;
    enabled: z.ZodBoolean;
    sink: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"sink">;
        valueType: z.ZodLiteral<"boolean">;
    }, z.core.$strip>>;
    source: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"boolean">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export type ToggleState = z.input<typeof toggleStateZ>;
/** Methods schema for Toggle RPC */
export declare const toggleMethodsZ: {
    toggle: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
};
interface InternalState {
    source: telem.BooleanSource;
    sink: telem.BooleanSink;
    addStatus: status.Adder;
    stopListening: destructor.Destructor;
    staleness: staleness.Registration;
}
export declare class Toggle extends aether.Leaf<typeof toggleStateZ, InternalState, typeof toggleMethodsZ> implements diagram.Element, aether.HandlersFromSchema<typeof toggleMethodsZ> {
    static readonly TYPE = "Toggle";
    static readonly METHODS: {
        toggle: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    static readonly z: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stale: z.ZodDefault<z.ZodBoolean>;
        enabled: z.ZodBoolean;
        sink: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
        source: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stale: z.ZodDefault<z.ZodBoolean>;
        enabled: z.ZodBoolean;
        sink: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
        source: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    methods: {
        toggle: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    afterUpdate(ctx: aether.Context): void;
    toggle(): void;
    private updateEnabledState;
    afterDelete(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=toggle.d.ts.map