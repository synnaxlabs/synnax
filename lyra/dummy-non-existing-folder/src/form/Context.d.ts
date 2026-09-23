import { type destructor } from "@synnaxlabs/x";
import { type z } from "zod";
import { type State } from "./state";
import { type Status } from "../status";
export interface RemoveFunc {
    (path: string): void;
}
/** Options for {@link SetFunc}. */
export interface SetOptions {
    /** Whether to call the form's `onChange`. Defaults to true. */
    notifyOnChange?: boolean;
    /** Whether the write counts as a user edit. Defaults to true. */
    markTouched?: boolean;
}
export interface SetFunc {
    (path: string, value: unknown, options?: SetOptions): void;
}
export interface Listener {
    (): void;
}
export interface BindFunc {
    (props: Listener): destructor.Destructor;
}
/** Whether the form takes edits, or renders flat and inert. */
export type Mode = "normal" | "preview";
/** The form API. Every field hook and component reads it from context. */
export interface ContextValue<Z extends z.ZodType = z.ZodType> {
    mode: Mode;
    /** Subscribes to every change in the form, returning the unsubscribe. */
    bind: BindFunc;
    /** Writes the value at a dot-separated path, validating and notifying. */
    set: SetFunc;
    /** Reads the state at a path: its value, status, and whether it is required. */
    get: typeof State.prototype.getState;
    /** Restores the initial values, or the given ones. */
    reset: (values?: z.infer<Z>) => void;
    remove: RemoveFunc;
    /** Reads the whole value tree. It does not re-render the caller. */
    value: () => z.infer<Z>;
    /** Validates the whole form, or one subtree, writing statuses onto the fields. */
    validate: (path?: string) => boolean;
    validateAsync: (path?: string) => Promise<boolean>;
    has: (path: string) => boolean;
    setStatus: typeof State.prototype.setStatus;
    clearStatuses: () => void;
    /** Takes the current values as the baseline, clearing the touched flag. */
    setCurrentStateAsInitialValues: () => void;
    getStatuses: () => Status.Crude[];
}
declare const Context: import("react").Context<ContextValue<z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>> | null>;
export { Context };
/**
 * @returns the enclosing form's {@link ContextValue}, or the override when one is given.
 * @throws {Error} if there is neither an enclosing form nor an override.
 */
export declare const useContext: <Z extends z.ZodType = z.ZodType>(override?: ContextValue<Z>, funcName?: string) => ContextValue<Z>;
//# sourceMappingURL=Context.d.ts.map