import { status } from "@synnaxlabs/client";
import { z } from "zod";
import { aether } from "../../aether/aether";
import { type Adder, type AsyncErrorHandler, type ErrorHandler } from "./errorHandler";
export declare const aggregatorStateZ: z.ZodObject<{
    statuses: z.ZodArray<status.StatusZodObject<z.ZodNever, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
}, z.core.$strip>;
export interface AggregatorState extends z.infer<typeof aggregatorStateZ> {
}
export declare class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
    static readonly TYPE: string;
    schema: z.ZodObject<{
        statuses: z.ZodArray<status.StatusZodObject<z.ZodNever, z.ZodEnum<{
            disabled: "disabled";
            error: "error";
            info: "info";
            loading: "loading";
            success: "success";
            warning: "warning";
        }>>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    private add;
}
export declare const useAdder: (ctx: aether.Context) => Adder;
export declare const useOptionalAdder: (ctx: aether.Context) => Adder;
export declare const useErrorHandler: (ctx: aether.Context) => ErrorHandler;
export declare const useAsyncErrorHandler: (ctx: aether.Context) => AsyncErrorHandler;
export declare const REGISTRY: aether.ComponentRegistry;
//# sourceMappingURL=aggregator.d.ts.map