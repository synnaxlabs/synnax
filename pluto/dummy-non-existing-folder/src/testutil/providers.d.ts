import { type Instrumentation } from "@synnaxlabs/alamos";
import { type z } from "zod";
import { type aether } from "../aether/aether";
import { aetherTest } from "../aether/test";
import { alamos } from "../alamos/aether";
import { status } from "../status/aether";
import { synnax } from "../synnax/aether";
import { telem } from "../telem/aether";
import { theming } from "../theming/aether";
import { type render } from "../vis/render";
import { canvasTest } from "../vis/render/test";
import { staleness } from "../vis/staleness/aether";
/**
 * Toggleable provider stack for Synnax test utilities. Each provider defaults to `true`
 * (mounted with harness defaults) except `render`, which is off until a recorder is
 * requested. Set a provider to `false` to drop it from the tree, or pass a config
 * object to override its initial state. The providers form a fixed nesting order
 * (`alamos → status → synnax → theming → telem → staleness → render`); disabling one
 * mounts the next directly under the previous enabled provider. A component that reads
 * context from a provider it disabled will not find it.
 */
export interface ProviderOptions {
    /** Alamos instrumentation provider. Defaults on. */
    alamos?: false | z.input<typeof alamos.providerStateZ>;
    /** Status aggregator. Defaults on with no statuses. */
    status?: false | z.input<typeof status.aggregatorStateZ>;
    /** Synnax client provider. Defaults on with no client. */
    synnax?: false | z.input<typeof synnax.Provider.stateZ>;
    /** Theming provider. Defaults on with `theme.SYNNAX_LIGHT`. */
    theming?: false | z.input<typeof theming.Provider.z>;
    /** Telemetry provider. Defaults on with `TestFactory` + `NoopFactory`. */
    telem?: false | {
        factories?: telem.Factory[];
    };
    /** Staleness provider. Defaults on with a 250ms sweep. */
    staleness?: false | z.input<typeof staleness.Provider.z>;
    /** Canvas render context. Off by default. `true` injects a fresh recorder; pass a
     * {@link canvasTest.Recorder} to supply your own and assert on it afterward, or any
     * other `render.Context`-shaped value to draw through it. */
    render?: boolean | canvasTest.Recorder | render.Context;
    /** Extra component types to register on the worker tree. */
    registry?: aether.ComponentRegistry;
    /** Instrumentation for the worker driver. */
    instrumentation?: Instrumentation;
}
/** Provider instances mounted by {@link buildStack}, by name. A field is `null` when its
 * provider was disabled. */
export interface MountedProviders {
    alamos: alamos.Provider | null;
    status: status.Aggregator | null;
    synnax: synnax.Provider | null;
    theming: theming.Provider | null;
    telem: aether.Composite<typeof telem.providerStateZ> | null;
    staleness: staleness.Provider | null;
    render: canvasTest.RenderProvider | null;
}
/** Result of {@link buildStack}: the worker driver, the path where the deepest provider
 * sits (descendants mount under it), the mounted provider instances, and the render
 * recorder if one was created or supplied. The recorder is null when the caller
 * supplied a plain render context instead. */
export interface BuiltStack {
    driver: aetherTest.Driver;
    basePath: string[];
    providers: MountedProviders;
    recorder: canvasTest.Recorder | null;
}
/**
 * Build a worker-side Synnax provider stack from {@link ProviderOptions}, on top of a
 * fresh {@link aetherTest.Driver}. Shared by `render` (main side) and `renderAether`
 * (worker side); most tests should call one of those, not this directly.
 */
export declare const buildStack: (options?: ProviderOptions) => BuiltStack;
//# sourceMappingURL=providers.d.ts.map