import { query, type Synnax as Client } from "@synnaxlabs/client";
export interface RetrieveParams<Query extends query.Params> {
    client: Client;
    query: Query;
}
interface UseMemoQuery {
    <Query extends query.Params>(q: Query, normalize?: (query: Query) => Query): Query;
    <Query extends query.Params>(q: Query | null, normalize?: (query: Query) => Query): Query | null;
}
/** Normalizes the query and stabilizes its identity across renders. */
export declare const useMemoQuery: UseMemoQuery;
/**
 * Fetch dedup and settled answers for queries the domain client does not cache. Entries
 * persist until the next fetch of the same query replaces them; an answer the cache
 * serves never populates `settled`. Scoped per client so a settled error never outlives
 * the client whose fetch produced it, and settled errors are dropped when the
 * connection epoch advances. Capped: write through {@link setSettled}, which evicts the
 * oldest entry past the cap.
 */
export interface LocalCache<Data> {
    epoch: number;
    inFlight: Map<string, Promise<Data>>;
    settled: Map<string, {
        data: Data;
    } | {
        error: Error;
    }>;
}
/**
 * Inserts a settled entry, evicting the oldest-inserted one once the cap is
 * reached. Settled data pins its answer in memory for the client's lifetime, so an
 * unbounded map leaks across a long session's distinct queries. An evicted entry
 * costs a refetch on the next read, never correctness.
 */
export declare const setSettled: <Data>(local: LocalCache<Data>, hash: string, entry: {
    data: Data;
} | {
    error: Error;
}) => void;
export declare const localFor: <Data>(locals: WeakMap<Client, LocalCache<Data>>, client: Client) => LocalCache<Data>;
export interface FetchErrorParams<Data> {
    cause: unknown;
    error: Error;
    hash: string;
    local: LocalCache<Data>;
}
export interface EnsureFetchParams<Query extends query.Params, Data> {
    name: string;
    retrieve: (params: RetrieveParams<Query>) => Promise<Data>;
    /**
     * Reports whether the fetched answer lands somewhere the next render reads.
     * An answer that does not is kept in `settled` instead.
     */
    getCached?: (params: RetrieveParams<Query>) => unknown;
    /**
     * Offers a failed fetch a second chance, returning a promise that supersedes
     * the failure or null to settle it.
     */
    onFetchError?: (params: RetrieveParams<Query>, errorParams: FetchErrorParams<Data>) => Promise<Data> | null;
    local: LocalCache<Data>;
}
/** Joins the query's in-flight fetch, starting one when none exists. */
export declare const ensureFetch: <Query extends query.Params, Data>(params: RetrieveParams<Query>, { name, retrieve, getCached, onFetchError, local }: EnsureFetchParams<Query, Data>) => Promise<Data>;
export interface PendingFetch<Data> {
    /** The promise the attempt being replayed suspended on, if there is one. */
    promise: Promise<Data> | null;
    /** Records the promise this attempt suspends on. */
    set: (promise: Promise<Data>) => void;
}
/**
 * Remembers the promise an attempt suspends on until the render commits. React ends
 * a replayed component's recorded hook list at its `use` call, so an attempt that
 * suspended has to resume through the same promise. Reaching the answer any other
 * way leaves every hook after the read reading a list that has already run out.
 */
export declare const usePendingFetch: <Query extends query.Params, Data>(q: Query | null) => PendingFetch<Data>;
export declare const suspendOnFetch: <Query extends query.Params, Data>(params: RetrieveParams<Query>, fetchParams: EnsureFetchParams<Query, Data>, pending: PendingFetch<Data>) => Data;
export {};
//# sourceMappingURL=suspend.d.ts.map