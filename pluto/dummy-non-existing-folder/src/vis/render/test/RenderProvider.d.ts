import { z } from "zod";
import { aether } from "../../../aether/aether";
export declare const renderProviderStateZ: z.ZodObject<{
    context: z.ZodAny;
}, z.core.$strip>;
/** Thin Composite that injects a `render.Context`-shaped value into the parent context
 * under the key consumed by `render.Context.use`. Lets a test recorder (or any
 * duck-typed render context) flow into the aether tree without a real `Canvas`
 * component, which depends on WebGL and OffscreenCanvas APIs jsdom does not provide. */
export declare class RenderProvider extends aether.Composite<typeof renderProviderStateZ> {
    static readonly TYPE = "render.test.RenderProvider";
    static readonly stateZ: z.ZodObject<{
        context: z.ZodAny;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        context: z.ZodAny;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
}
//# sourceMappingURL=RenderProvider.d.ts.map