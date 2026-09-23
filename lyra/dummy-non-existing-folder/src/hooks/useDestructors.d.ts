import { type destructor } from "@synnaxlabs/x";
/** The handle returned by {@link useDestructors}. */
export interface UseDestructorsReturn {
    /** Runs every collected destructor and empties the collection. */
    cleanup: () => void;
    /** Adds destructors to the collection. Undefined is ignored. */
    set: (destructors: destructor.Destructor | destructor.Destructor[] | undefined) => void;
}
/**
 * Collects destructors across renders and runs them on unmount. Use it where teardown
 * is registered outside the render pass (an event subscription made in a callback) and
 * so cannot ride an effect's return value.
 */
export declare const useDestructors: () => UseDestructorsReturn;
//# sourceMappingURL=useDestructors.d.ts.map