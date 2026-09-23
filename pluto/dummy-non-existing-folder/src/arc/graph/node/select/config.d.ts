import z from "zod/v4";
export declare const configZ: z.ZodObject<{
    type: z.ZodLiteral<"select">;
}, z.core.$strip>;
export interface Config extends z.infer<typeof configZ> {
}
export declare const defaultConfig: () => Config;
//# sourceMappingURL=config.d.ts.map