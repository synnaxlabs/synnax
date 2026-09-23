import { z } from "zod";
import { aether } from "../../../aether/aether";
import { render } from "../../render";
export declare const canvasStateZ: z.ZodObject<{
    dpr: z.ZodNumber;
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
    bootstrap: z.ZodDefault<z.ZodBoolean>;
    bootstrapped: z.ZodDefault<z.ZodBoolean>;
    glCanvas: z.ZodOptional<z.ZodAny>;
    upper2dCanvas: z.ZodOptional<z.ZodAny>;
    lower2dCanvas: z.ZodOptional<z.ZodAny>;
    os: z.ZodUnion<[z.ZodEnum<{
        Linux: "Linux";
        Windows: "Windows";
        macOS: "macOS";
    }>, z.ZodPipe<z.ZodEnum<{
        linux: "linux";
        macos: "macos";
        windows: "windows";
    }>, z.ZodTransform<"Linux" | "Windows" | "macOS", "linux" | "macos" | "windows">>]>;
}, z.core.$strip>;
export declare class Canvas extends aether.Composite<typeof canvasStateZ> {
    static readonly TYPE = "Canvas";
    schema: z.ZodObject<{
        dpr: z.ZodNumber;
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
        bootstrap: z.ZodDefault<z.ZodBoolean>;
        bootstrapped: z.ZodDefault<z.ZodBoolean>;
        glCanvas: z.ZodOptional<z.ZodAny>;
        upper2dCanvas: z.ZodOptional<z.ZodAny>;
        lower2dCanvas: z.ZodOptional<z.ZodAny>;
        os: z.ZodUnion<[z.ZodEnum<{
            Linux: "Linux";
            Windows: "Windows";
            macOS: "macOS";
        }>, z.ZodPipe<z.ZodEnum<{
            linux: "linux";
            macos: "macos";
            windows: "windows";
        }>, z.ZodTransform<"Linux" | "Windows" | "macOS", "linux" | "macos" | "windows">>]>;
    }, z.core.$strip>;
    renderCtx: render.Context | null;
    afterUpdate(ctx: aether.Context): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
//# sourceMappingURL=canvas.d.ts.map