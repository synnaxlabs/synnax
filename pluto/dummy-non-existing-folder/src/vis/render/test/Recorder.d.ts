import { type border, box, type destructor, type dimensions, scale, xy } from "@synnaxlabs/x";
import { type render } from "..";
/** A single recorded canvas method call or property assignment. */
export interface Call {
    op: string;
    args: unknown[];
}
/** Recording analog of `SugaredOffscreenCanvasRenderingContext2D`. It is itself the
 * drawing surface — every method call and property assignment is captured in
 * {@link calls} in encounter order — so it can stand in directly wherever production
 * reads `render.Context.lower2d` / `upper2d` (e.g. constructing a `Draw2D`). */
export interface RecordingCanvas {
    /** Recorded method calls and property sets, in encounter order. */
    calls: Call[];
    /** Restrict drawing to a region; records the call and returns a no-op destructor. */
    scissor(region: box.Box, overScan?: xy.XY, radius?: border.CrudeRadius): destructor.Destructor;
    /** Records the call and returns this same recording surface, so chained draw calls
     * land on the same {@link calls} list (production returns a scaled sub-context). */
    applyScale(scale: scale.XY): RecordingCanvas;
    /** Records the call and returns a deterministic size derived from the label length, so
     * width-dependent layout logic under test gets stable, varied inputs. */
    textDimensions(label: string, options?: unknown): dimensions.Dimensions;
    [op: string]: unknown;
}
/** A single `scissor()` call recorded on the recorder. */
export interface ScissorCall {
    region: box.Box;
    overScan: xy.XY;
    canvases: render.CanvasVariant[];
}
/** A single `erase()` call recorded on the recorder. */
export interface EraseCall {
    region: box.Box;
    overScan: xy.Crude;
    canvases: render.CanvasVariant[];
}
/** A single `loop.set()` call recorded on the recorder. */
export interface LoopCall {
    args: unknown[];
}
/** Duck-typed render context that records every component-visible call.
 *
 * Pass as the `render` option to `renderAether` (or `render`) to make a component under
 * test see this recorder as its `render.Context`. The 2D canvases are themselves
 * recording drawing surfaces, so components that draw (via `Draw2D`) work unchanged;
 * recorded data is available on the recorder for assertion afterward. */
export declare class Recorder {
    readonly upper2d: RecordingCanvas;
    readonly lower2d: RecordingCanvas;
    readonly gl: RecordingCanvas;
    /** Calls to `Recorder.scissor` in encounter order. */
    readonly scissorCalls: ScissorCall[];
    /** Calls to `Recorder.erase` in encounter order. */
    readonly eraseCalls: EraseCall[];
    /** Calls to `Recorder.loop.set` in encounter order. */
    readonly loopCalls: LoopCall[];
    region: box.Box;
    dpr: number;
    readonly loop: {
        set: (...args: unknown[]) => void;
    };
    constructor();
    /** Resets every recording — the per-canvas calls and the scissor/erase/loop lists.
     * Region and dpr are left as-is. Use between phases of a test when you only care
     * about calls made since `clear`. */
    clear(): void;
    scissor(region: box.Box, overScan?: xy.XY, canvases?: render.CanvasVariant[]): destructor.Destructor;
    erase(region: box.Box, overScan?: xy.Crude, ...canvases: render.CanvasVariant[]): void;
    resize(region: box.Box, dpr: number): void;
    get aspect(): number;
    scaleRegion(_b: box.Box): scale.XY;
}
//# sourceMappingURL=Recorder.d.ts.map