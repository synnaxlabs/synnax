import { alamos } from "@synnaxlabs/alamos";
import { type status } from "../../status/aether";
import { type CanvasVariant } from "./context";
/**
 * A function that executes the render in the loop. This function can return an
 * optional cleanup that is executed before the next render. This cleanup function will
 * be passed the signature of the previous render request.
 */
export interface Renderer {
    (): Cleanup | void;
}
/**
 * A request to render a component in the aether visualization tree. Submit a complete
 * version of this request to the {@link Loop} to render a component.
 */
export interface Request {
    /**
     * A key identifying the component requesting the render. This helps to prevent
     * duplicate renders for the same component from being executed.
     */
    key: string;
    /**
     * A priority ("high" or "low") for the render. High priority renders that have
     * an equal or greater number of canvases will replace low priority renders already
     * requested.
     */
    priority: Priority;
    /**
     * A list of canvases that the component is requesting to render to. This provides
     * information to cleanup functions about which canvases to clear. The component should
     * ONLY render to these canvases, otherwise the cleanup function may unnecessarily
     * clear canvases that should persist.
     */
    canvases: CanvasVariant[];
    /**
     * An async function that performs the render. This function can return an optional
     * cleanup that is executed before the next render. This cleanup function will be
     * passed the signature of the previous render request.
     */
    render: Renderer;
}
/**
 * A cleanup function that receives the request from the previous render. Cleanup
 * functions should clear canvases and other resources that need to be freed from
 * the previous render.
 */
export type Cleanup = (req: Request) => void;
export type Priority = "high" | "low";
interface LoopParams {
    handleError: status.ErrorHandler;
    afterRender?: () => void;
    instrumentation?: alamos.Instrumentation;
}
/**
 * Implements the main rendering loop for Synnax's aether components, accepting requests
 * into a queue and rendering them in sync with the browser animation frame.
 */
export declare class Loop {
    /** Stores the current requests for rendering. */
    private readonly requests;
    /** Stores render cleanup functions for clearing canvases and other resources. */
    private readonly cleanup;
    /** A callback to run after each render call. */
    private readonly afterRender?;
    /** Instrumentation for logging, tracing, metrics, etc. */
    private readonly instrumentation;
    /** A function to add status to the status bar. */
    private readonly handleError;
    constructor({ afterRender, instrumentation, handleError, }: LoopParams);
    /**
     * Sets a new request in the queue according to a set of rules:
     *
     * 1. If no request with the same key exists, add the request to the queue.
     * 2. If a request with the same key exists, replace it if the new request has a
     * greater or equal priority and a greater or equal number of canvases that are
     * being rendered to.
     */
    set(req: Request): void;
    /** Execute the render. */
    private render;
    private runCleanupsSync;
    private renderSync;
    /** Starts the rendering loop. */
    private start;
}
export {};
//# sourceMappingURL=loop.d.ts.map