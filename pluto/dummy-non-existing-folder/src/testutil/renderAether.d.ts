import { state } from "@synnaxlabs/x";
import { type z } from "zod";
import { type aether } from "../aether/aether";
import { type MountedProviders, type ProviderOptions } from "./providers";
/** Initial state for a child mounted under the component under test. */
export interface MountChild {
    type: string;
    state: state.State;
}
/** Constructor + static-schema shape every component passed to {@link renderAether} must
 * satisfy. The state schema is read for typing only (the Leaf parses incoming state
 * itself), and is accepted under either the `stateZ` or `z` static — both conventions
 * exist in the codebase. */
export type ComponentClass = {
    TYPE: string;
    new (props: aether.ComponentConstructorProps): aether.Component;
} & ({
    stateZ: z.ZodType<state.State>;
} | {
    z: z.ZodType<state.State>;
});
/** The Zod state schema declared on a {@link ComponentClass}, read from either the `z`
 * or `stateZ` static. */
export type SchemaOf<C extends ComponentClass> = C extends {
    z: infer Z extends z.ZodType<state.State>;
} ? Z : C extends {
    stateZ: infer Z extends z.ZodType<state.State>;
} ? Z : never;
/** Handle returned by {@link renderAether}. `component` is typed as the concrete class
 * instance, so its public methods and fields are callable directly and fully typed —
 * e.g. `h.component.onMouseDown()` — with no casts. */
export interface Handle<C extends ComponentClass> {
    /** The component under test, typed as its concrete class. Call its methods directly
     * to drive behavior. */
    readonly component: InstanceType<C>;
    /** Current parsed state of the component under test. */
    readonly state: z.infer<SchemaOf<C>>;
    /** Replace the component's state. Accepts a value or a `(prev) => next` setter; the
     * result is parsed against the component's schema. */
    setState(next: state.SetArg<z.infer<SchemaOf<C>>>): void;
    /** Create or replace a child of the component under test. `type` must be registered. */
    setChildState(key: string, type: string, childState: state.State): void;
    /** Delete a child of the component under test. */
    deleteChild(key: string): void;
    /** Look up a child instance by key. The `T` cast is unchecked; align it with the
     * registered class. */
    child<T extends aether.Component = aether.Component>(key: string): T;
    /** Provider instances in the stack, exposed for direct manipulation. */
    readonly providers: MountedProviders;
    /** Tear down the tree early. Mounts are torn down automatically after each test, so
     * this is only needed to assert teardown behavior or to release before the test ends.
     * Idempotent. */
    unmount(): void;
}
/** Options for {@link renderAether}. `state` is typed as the schema's input, so
 * fields with Zod defaults may be omitted. Provider toggles come from
 * {@link ProviderOptions}. */
export interface RenderAetherOptions<C extends ComponentClass> extends ProviderOptions {
    state: z.input<SchemaOf<C>>;
    /** Instance key for the component under test. Defaults to the component's TYPE. */
    key?: string;
    /** Initial children to mount under the component under test. */
    children?: Record<string, MountChild>;
}
/**
 * Mount an aether component on the worker side, wrapped in a Synnax provider stack.
 * Builds the chain `Root → [alamos → status → synnax → theming → telem → render] →
 * <component>`, including only the providers enabled in {@link ProviderOptions} (all on
 * by default except `render`). Every level runs its real `afterUpdate` lifecycle and
 * propagates context the way production does; the test never wires `parent` by hand.
 * The mount is torn down automatically after each test. To assert on draw calls, create
 * a recorder and pass it as `render`, then assert on that reference — the same fixture
 * pattern as `telemTest.sink()` / `source()`. For tests that exercise the React +
 * worker boundary, use `render` instead.
 */
export declare const renderAether: <C extends ComponentClass>(Component: C, options: RenderAetherOptions<C>) => Handle<C>;
//# sourceMappingURL=renderAether.d.ts.map