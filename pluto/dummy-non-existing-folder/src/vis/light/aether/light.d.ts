import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
import { type diagram } from "../../diagram/aether";
import { staleness } from "../../staleness/aether";
export declare const stateZ: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stale: z.ZodDefault<z.ZodBoolean>;
    enabled: z.ZodBoolean;
    source: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"boolean">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export interface State extends z.input<typeof stateZ> {
}
interface InternalState {
    source: telem.BooleanSource;
    stopListening: destructor.Destructor;
    staleness: staleness.Registration;
}
export declare class Light extends aether.Leaf<typeof stateZ, InternalState> implements diagram.Element {
    static readonly TYPE = "Light";
    static readonly z: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stale: z.ZodDefault<z.ZodBoolean>;
        enabled: z.ZodBoolean;
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
        source: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"boolean">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    private updateEnabledState;
    afterDelete(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=light.d.ts.map