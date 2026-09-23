import { query } from "@synnaxlabs/client";
import { type state } from "@synnaxlabs/x";
import { type flux } from "./aether";
import { type Tombstone } from "./errors";
import { type Result } from "./result";
import { type RetrieveParams } from "./suspend";
export { type RetrieveParams };
/**
 * Binds a query definition onto its domain client's read surface: fetch =
 * `retrieve`, live updates = `onChange`, snapshot = `getCached`. Flux holds no
 * cache of its own; queries without `onChange` and `getCached` fetch on every
 * mount and never receive live updates.
 */
export interface CreateRetrieveParams<Query extends query.Params, Data extends query.Data> extends flux.Definition<Query, Data> {
    /**
     * Holds the previous answer whenever the next one compares equal, so readers
     * re-render only on changes the answer expresses. Defaults to element-wise
     * identity for list answers, which is what a `getCached` that composes entries
     * out of the domain client's tables needs. Override only for an answer whose
     * equality the default reads as a change.
     */
    equal?: (prev: Data, next: Data) => boolean;
    /**
     * Canonicalizes the caller's query before anything reads it: `retrieve`,
     * `onChange`, and `getCached` all receive the one normalized, identity-stable
     * object. Merge defaults here instead of at each callback, where a per-call
     * spread would mint a fresh object and miss the client's query memos. Must
     * preserve fields it does not set, so selector-only extensions survive.
     */
    normalizeQuery?: <Q extends Query>(query: Q) => Q;
    /**
     * Holds a not-found pending for a short wait instead of settling it, for documents a
     * reference can reach a reader ahead of: a panel tab minted in another window names
     * its view before that view's create broadcast lands. Every other reader shows
     * absence at once. Requires `onChange` and `getCached`.
     */
    awaitCreation?: boolean;
    /**
     * Asks again, with a widening wait, when a fetch fails because the Core could not be
     * reached. For reads a surface cannot render without: a settled failure holds until
     * something invalidates it, so one unlucky moment reads as a lasting one.
     */
    retryUnreachable?: boolean;
}
export interface Use<Query extends query.Params, Data extends state.State> {
    (query: Query): Data;
}
export interface UseEnsure<Query extends query.Params> {
    (query: Query): void;
}
/**
 * Returns a callback discarding a query's settled answer so the next suspending
 * read fetches again. A settled failure re-throws on every render, so resetting
 * an error boundary alone lands straight back on it.
 */
export interface UseInvalidate<Query extends query.Params> {
    (): (query: Query) => void;
}
export interface UseTombstone<Query extends query.Params> {
    (query: Query): Tombstone | null;
}
/**
 * A warm-cache projected read: returns the selected slice of the query's cached
 * answer and re-renders only when that slice changes. Never suspends and never
 * fetches; a parent must have retrieved the query (see useEnsure).
 * @throws {NotFoundError} when nothing is cached for the query.
 * @throws {DeletedError} when the cached answer is a tombstone.
 * @throws {DisconnectedError} when no Core is connected.
 */
export interface UseSelect<Query extends query.Params, Selected> {
    (query: Query): Selected;
}
/**
 * A reactive read for callers that must handle loading and failure themselves, where
 * suspension is illegal or absence is a state to render rather than an error. Serves
 * the cached answer, subscribes for changes, and kicks a deduped background fetch on a
 * cold miss. Never suspends and never throws: the variant reports loading, success, and
 * failure, a null query or absent client reads as disabled, and an answer the domain
 * cache never holds is served once, from the fetch. Reach for `use` wherever the caller
 * may suspend.
 */
export interface UseResult<Query extends query.Params, Data extends state.State> {
    (query: Query | null): Result<Data>;
}
/**
 * Mints a {@link UseSelect} sharing the definition's cache wiring. ExtendedQuery adds
 * selector-only fields (a node key, a tab key) the select projection needs; the cache
 * layer ignores them when addressing the record.
 */
export interface CreateSelector<Query extends query.Params, Data extends query.Data> {
    <Selected, ExtendedQuery extends Query = Query>(select: (data: Data, query: ExtendedQuery) => Selected, equal?: (a: Selected, b: Selected) => boolean): UseSelect<ExtendedQuery, Selected>;
}
/**
 * Mints a {@link UseResult}-shaped hook that re-renders only when the selected slice
 * of the answer changes: {@link UseSelect}'s render gating with {@link UseResult}'s
 * contract — fetch on a cold miss, never suspend, never throw. For absence-tolerant
 * narrow readers; warm-cache readers under a parent retrieve use
 * {@link CreateSelector}.
 */
export interface CreateResultSelector<Query extends query.Params, Data extends query.Data> {
    <Selected extends state.State, ExtendedQuery extends Query = Query>(select: (data: Data, query: ExtendedQuery) => Selected, equal?: (a: Selected, b: Selected) => boolean): UseResult<ExtendedQuery, Selected>;
}
export interface CreateRetrieveReturn<Query extends query.Params, Data extends state.State> {
    use: Use<Query, Data>;
    useEnsure: UseEnsure<Query>;
    useResult: UseResult<Query, Data>;
    useInvalidate: UseInvalidate<Query>;
    useTombstone: UseTombstone<Query>;
    createSelector: CreateSelector<Query, Data>;
    createResultSelector: CreateResultSelector<Query, Data>;
}
export declare const createRetrieve: <Query extends query.Params, Data extends query.Data>(createParams: CreateRetrieveParams<Query, Data>) => CreateRetrieveReturn<Query, Data>;
//# sourceMappingURL=retrieve.d.ts.map