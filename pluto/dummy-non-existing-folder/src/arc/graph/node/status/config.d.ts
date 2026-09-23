import z from "zod/v4";
export declare const configZ: z.ZodObject<{
    type: z.ZodLiteral<"status.set">;
    key_or_name: z.ZodString;
    variant: z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>;
    message: z.ZodString;
}, z.core.$strip>;
export interface Config extends z.infer<typeof configZ> {
}
export declare const defaultConfig: () => Config;
//# sourceMappingURL=config.d.ts.map