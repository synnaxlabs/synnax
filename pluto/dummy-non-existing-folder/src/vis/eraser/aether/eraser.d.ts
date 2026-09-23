import { z } from "zod";
import { aether } from "../../../aether/aether";
import { render } from "../../render";
export declare const eraserStateZ: z.ZodObject<{
    region: z.ZodObject<{
        one: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        two: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        root: z.ZodObject<{
            x: z.ZodEnum<{
                left: "left";
                right: "right";
            }>;
            y: z.ZodEnum<{
                bottom: "bottom";
                top: "top";
            }>;
        }, z.core.$strip>;
    }, z.core.$strip>;
    enabled: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
interface InternalState {
    renderCtx: render.Context | null;
}
/**
 * Erases its region from the canvases on lifecycle changes. When no canvas render
 * context exists (a canvas-less mount), there is nothing to erase and Eraser no-ops.
 */
export declare class Eraser extends aether.Leaf<typeof eraserStateZ, InternalState> {
    static readonly TYPE = "eraser";
    schema: z.ZodObject<{
        region: z.ZodObject<{
            one: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            two: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            root: z.ZodObject<{
                x: z.ZodEnum<{
                    left: "left";
                    right: "right";
                }>;
                y: z.ZodEnum<{
                    bottom: "bottom";
                    top: "top";
                }>;
            }, z.core.$strip>;
        }, z.core.$strip>;
        enabled: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    renderOnLifecycleChange(): void;
    render(): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=eraser.d.ts.map