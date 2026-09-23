import { type Instrumentation } from "@synnaxlabs/alamos";
import { type state } from "@synnaxlabs/x";
import { aether } from "../aether";
/** Key of the synthetic root mounted by {@link createDriver}. All driver paths begin
 * with this segment. */
export declare const ROOT_KEY = "root";
/** Low-level handle over a worker-side aether tree. Drives state updates and deletes
 * against a real {@link aether.Root} backed by a mock comms pair, with no knowledge of
 * any particular provider. Higher-level Synnax test utilities build their provider
 * stacks on top of this. */
export interface Driver {
    /** Root of the worker-side tree. */
    root: aether.Root;
    /** Worker end of the mock comms pair. */
    workerSide: aether.WorkerComms;
    /** Main end of the mock comms pair, for bridging a React tree into this driver. */
    mainSide: aether.MainComms;
    /** Create or replace the component at `path`, constructing it from the registry if it
     * does not yet exist. `path` is absolute and begins with {@link ROOT_KEY}. */
    update(path: readonly string[], type: string, state: state.State): void;
    /** Delete the component at `path` and its descendants. */
    delete(path: readonly string[]): void;
    /** Look up a mounted component by absolute path. Throws if no component exists there. */
    find<T extends aether.Component = aether.Component>(path: readonly string[]): T;
}
/**
 * Create a {@link Driver} over a fresh worker-side aether tree using the given
 * registry. The driver constructs components lazily through the registry the same way
 * production does — the caller never wires `parent` by hand, so every level runs its
 * real `afterUpdate` lifecycle and propagates context normally.
 */
export declare const createDriver: (registry: aether.ComponentRegistry, instrumentation?: Instrumentation) => Driver;
//# sourceMappingURL=driver.d.ts.map