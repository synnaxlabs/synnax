import { alamos } from "@synnaxlabs/alamos";
import { z } from "zod";
import { aether } from "../../aether/aether";
export declare const providerStateZ: z.ZodObject<{
    include: z.ZodOptional<z.ZodArray<z.ZodString>>;
    exclude: z.ZodOptional<z.ZodArray<z.ZodString>>;
    level: z.ZodDefault<z.ZodEnum<{
        debug: "debug";
        error: "error";
        info: "info";
        warn: "warn";
    }>>;
}, z.core.$strip>;
export type ProviderState = z.input<typeof providerStateZ>;
export interface InternalState {
    ins: alamos.Instrumentation;
}
export declare class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = "alamos.Provider";
    schema: z.ZodObject<{
        include: z.ZodOptional<z.ZodArray<z.ZodString>>;
        exclude: z.ZodOptional<z.ZodArray<z.ZodString>>;
        level: z.ZodDefault<z.ZodEnum<{
            debug: "debug";
            error: "error";
            info: "info";
            warn: "warn";
        }>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
}
export declare const useInstrumentation: (ctx: aether.Context, name?: string) => alamos.Instrumentation;
export declare const REGISTRY: aether.ComponentRegistry;
//# sourceMappingURL=alamos.d.ts.map