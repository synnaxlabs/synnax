import { type actions, type query, type Synnax as Client } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
/** One or more actions aimed at the document with the given key. */
export interface DispatchInput<Key extends record.Key, Action> {
    key: Key;
    actions: Action | Action[];
}
/** Params for {@link createDispatch}. */
export interface CreateDispatchParams<Key extends record.Key, State extends query.Data, Action> {
    /** Selects the domain's dispatch surface off the client. */
    domain: (client: Client) => actions.Domain<Key, State, Action>;
    /**
     * Rewrites an action batch against the current document before replay and
     * send. Lives here rather than in the domain client when the rewrite needs
     * visualization-layer context (e.g. diagram geometry).
     */
    preprocess?: actions.Preprocess<State, Action>;
}
/** Return value for the `useDispatch` hook. */
export interface UseDispatchReturn<Key extends record.Key, Action> {
    /** Applies actions and reports a failure as an error status. */
    dispatch: (input: DispatchInput<Key, Action>) => void;
    /** Applies actions and resolves to whether they committed. */
    dispatchAsync: (input: DispatchInput<Key, Action>) => Promise<boolean>;
    /** Groups actions so undo takes them back as one step. */
    beginTransaction: (input: {
        key: Key;
        kind?: string;
    }) => actions.Transaction<Action>;
}
/** Return value for the `useUndo` hook. */
export interface UseUndoReturn {
    undo: () => void;
    canUndo: boolean;
}
/** Return value for the `useRedo` hook. */
export interface UseRedoReturn {
    redo: () => void;
    canRedo: boolean;
}
/** The hooks {@link createDispatch} builds for one domain. */
export interface CreateDispatchReturn<Key extends record.Key, Action> {
    useDispatch: () => UseDispatchReturn<Key, Action>;
    useUndo: (params: {
        key: Key;
    }) => UseUndoReturn;
    useRedo: (params: record.Keyed<Key>) => UseRedoReturn;
    useSingleDispatch: (params: record.Keyed<Key>) => (action: Action | Action[]) => void;
}
/**
 * Builds the dispatch, undo, and redo hooks for one editable document domain. Call it
 * once per domain, at module scope.
 */
export declare const createDispatch: <Key extends record.Key, State extends query.Data, Action>({ domain, preprocess, }: CreateDispatchParams<Key, State, Action>) => CreateDispatchReturn<Key, Action>;
//# sourceMappingURL=dispatch.d.ts.map