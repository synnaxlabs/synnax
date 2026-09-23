import { status } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../aether";
export declare const chipStatusDetailsZ: z.ZodDefault<z.ZodObject<{
    authority: z.ZodOptional<z.ZodInt>;
    valid: z.ZodOptional<z.ZodBoolean>;
}, z.core.$strip>>;
export declare const chipStateZ: z.ZodObject<{
    triggered: z.ZodBoolean;
    status: status.StatusZodObject<z.ZodDefault<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        valid: z.ZodOptional<z.ZodBoolean>;
    }, z.core.$strip>>, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>;
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
        valueType: z.ZodLiteral<"status">;
    }, z.core.$strip>>;
}, z.core.$strip>;
interface InternalState {
    source: telem.StatusSource<typeof chipStatusDetailsZ>;
    sink: telem.BooleanSink;
    stopListening: destructor.Destructor;
}
export declare class Chip extends aether.Leaf<typeof chipStateZ, InternalState> {
    static readonly TYPE = "Chip";
    schema: z.ZodObject<{
        triggered: z.ZodBoolean;
        status: status.StatusZodObject<z.ZodDefault<z.ZodObject<{
            authority: z.ZodOptional<z.ZodInt>;
            valid: z.ZodOptional<z.ZodBoolean>;
        }, z.core.$strip>>, z.ZodEnum<{
            disabled: "disabled";
            error: "error";
            info: "info";
            loading: "loading";
            success: "success";
            warning: "warning";
        }>>;
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
            valueType: z.ZodLiteral<"status">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    private updateEnabledState;
    afterDelete(): void;
    render(): void;
}
export {};
//# sourceMappingURL=chip.d.ts.map