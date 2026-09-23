import { type RenderHookOptions, type RenderHookResult, type RenderOptions as RTLRenderOptions, type RenderResult as RTLRenderResult } from "@testing-library/react";
import { type ReactElement } from "react";
import { type aether } from "../aether/aether";
import { type MountedProviders, type ProviderOptions } from "./providers";
import { type canvasTest } from "../vis/render/test";
/** Options for {@link render}: provider toggles plus standard RTL render options
 * (minus `wrapper`, which the harness owns). */
export interface RenderOptions extends ProviderOptions {
    rtl?: Omit<RTLRenderOptions, "wrapper">;
}
/** RTL render result plus references into the worker tree the harness mounted. */
export interface RenderResult extends RTLRenderResult {
    /** Root of the worker tree. Use `root.findChildAtPath(...)` to grab any mounted
     * component by path. */
    root: aether.Root;
    /** Provider instances mounted on the worker, by name. */
    providers: MountedProviders;
    /** The render recorder, if `render` was enabled; otherwise `null`. */
    recorder: canvasTest.Recorder | null;
}
/**
 * Renders a hook that suspends on a cold cache, resolving once its render commits.
 * RTL's own `renderHook` never commits a tree that suspends during the initial render,
 * leaving `result.current` null forever. Never call this inside an `act` scope: the
 * commit it waits on cannot land until that scope exits.
 */
export declare const renderHookSuspended: <Result, Props>(hook: (props: Props) => Result, options?: RenderHookOptions<Props>) => Promise<RenderHookResult<Result, Props>>;
/**
 * Render a React component with a Synnax provider stack already mounted on the worker
 * side, so the component's `Aether.use` calls register under the stack without the test
 * scaffolding a multi-level provider tree by hand. Use this for tests that exercise the
 * React + worker boundary (clicks, form input, dispatched actions). For pure
 * worker-side renderer testing, use `renderAether`.
 */
export declare const render: (ui: ReactElement, options?: RenderOptions) => RenderResult;
//# sourceMappingURL=render.d.ts.map