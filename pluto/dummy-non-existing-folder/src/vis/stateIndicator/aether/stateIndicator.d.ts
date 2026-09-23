import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
import { type diagram } from "../../diagram/aether";
import { staleness } from "../../staleness/aether";
export declare const stateZ: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stale: z.ZodDefault<z.ZodBoolean>;
    key: z.ZodDefault<z.ZodNullable<z.ZodString>>;
    options: z.ZodDefault<z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        value: z.ZodNumber;
    }, z.core.$strip>>>;
    source: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"number">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export interface State extends z.input<typeof stateZ> {
}
interface InternalState {
    source: telem.NumberSource;
    stopListening: destructor.Destructor;
    staleness: staleness.Registration;
}
export declare class StateIndicator extends aether.Leaf<typeof stateZ, InternalState> implements diagram.Element {
    static readonly TYPE = "StateIndicator";
    static readonly z: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stale: z.ZodDefault<z.ZodBoolean>;
        key: z.ZodDefault<z.ZodNullable<z.ZodString>>;
        options: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            value: z.ZodNumber;
        }, z.core.$strip>>>;
        source: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"number">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stale: z.ZodDefault<z.ZodBoolean>;
        key: z.ZodDefault<z.ZodNullable<z.ZodString>>;
        options: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            value: z.ZodNumber;
        }, z.core.$strip>>>;
        source: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"number">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    private updateMatchedOption;
    afterDelete(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=stateIndicator.d.ts.map