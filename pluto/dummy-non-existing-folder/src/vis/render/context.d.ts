import { box, type destructor, type runtime, scale, xy } from "@synnaxlabs/x";
import { type aether } from "../../aether/aether";
import { SugaredOffscreenCanvasRenderingContext2D } from "../draw2d/canvas";
import { Loop } from "./loop";
export type Canvas2DVariant = "upper2d" | "lower2d";
export type CanvasGLVariant = "gl";
export type CanvasVariant = Canvas2DVariant | CanvasGLVariant;
/**
 * A hybrid rendering context containing both 2D and WebGL canvases and contexts.
 * Implements several utility methods for correctly scaling the canvas, restricting
 * drawing to a region, and erasing portions of the canvas.
 */
export declare class Context {
    readonly glCanvas: OffscreenCanvas;
    /** The canvas element used by the 2D canvas. */
    readonly upper2dCanvas: OffscreenCanvas;
    /** The canvas element used by the 2D canvas. */
    readonly lower2dCanvas: OffscreenCanvas;
    /** The WebGL rendering context.  */
    readonly gl: WebGL2RenderingContext;
    /** A 2D canvas that sits below the WebGL canvas. */
    lower2d: SugaredOffscreenCanvasRenderingContext2D;
    /** A 2D canvas that sits above the WebGL canvas. */
    upper2d: SugaredOffscreenCanvasRenderingContext2D;
    /** The region the canvas occupies in pixel space */
    region: box.Box;
    /** The device pixel ratio of the canvas */
    dpr: number;
    /** queue render transitions onto the stack */
    readonly loop: Loop;
    /** See the @link{clear.Program} for why this is necessary. */
    private readonly clearProgram?;
    private readonly os;
    private static readonly CONTEXT_KEY;
    private readonly instrumentation;
    static create(ctx: aether.Context, glCanvas: OffscreenCanvas, lower2dCanvas: OffscreenCanvas, upper2dCanvas: OffscreenCanvas, os: runtime.OS): Context;
    private constructor();
    static useOptional(ctx: aether.Context): Context | null;
    static use(ctx: aether.Context): Context;
    update(ctx: aether.Context): void;
    /**
     * Resizes the canvas to the given region and device pixel ratio. Ensuring
     * that all drawing operations and viewports are scaled correctly.
     */
    resize(region: box.Box, dpr: number): void;
    private resizeCanvas;
    /** @returns the aspect ratio of the canvas. */
    get aspect(): number;
    /**
     * Takes the given box in PIXEL space and produces a transform
     * in CLIP space representing the sub-region represented by the box
     * in the canvas.
     */
    scaleRegion(b: box.Box): scale.XY;
    scissor(region: box.Box, overScan: xy.XY | undefined, canvases: CanvasVariant[]): destructor.Destructor;
    private scissorGL;
    erase(region: box.Box, overscan?: xy.Crude, ...canvases: CanvasVariant[]): void;
    private eraseGL;
    private eraseCanvas;
}
//# sourceMappingURL=context.d.ts.map