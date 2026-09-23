import { query } from "@synnaxlabs/client";
import { compare, type CrudeTimeSpan, type destructor, type record, state } from "@synnaxlabs/x";
import { type Result } from "./result";
import { type CreateRetrieveParams, type RetrieveParams } from "./retrieve";
/** Reads one entry, or many at once, out of a list query's results. */
export interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
    (key: K): E | undefined;
    (keys: K[]): E[];
}
/** Options for a list retrieval. Append for paging, replace for a new search. */
export interface AsyncListOptions extends query.FetchOptions {
    mode?: "append" | "replace";
}
/** Return value for a list hook. Spread it into a {@link List.Frame}. */
export type UseListReturn<Query extends query.Params, K extends record.Key, E extends record.Keyed<K>> = Omit<Result<K[]>, "data"> & {
    retrieve: (query: state.SetArg<Query, Partial<Query>>, options?: AsyncListOptions) => void;
    retrieveAsync: (query: state.SetArg<Query, Partial<Query>>, options?: AsyncListOptions) => Promise<void>;
    data: K[];
    /**
     * Whether the list has an answer. An unanswered list has the same empty data as a
     * genuinely empty one, so anything that speaks for the absence of items (empty
     * content, an empty-state action) must wait for this.
     */
    answered: boolean;
    getItem: GetItem<K, E>;
    subscribe: (callback: () => void, key: K) => destructor.Destructor;
};
export interface RetrieveByKeyParams<Query extends query.Params, K extends record.Key> extends Omit<RetrieveParams<Query>, "query"> {
    query: Partial<Query>;
    key: K;
}
export interface CreateListParams<Query extends query.Params, K extends record.Key, E extends record.Keyed<K>> extends CreateRetrieveParams<Query, E[]> {
    sort?: compare.Comparator<E>;
    retrieveByKey: (params: RetrieveByKeyParams<Query, K>) => Promise<E | undefined>;
    /**
     * Live updates for items fetched through retrieveByKey. Page members get
     * their updates from `onChange`; this covers lookups outside any page.
     */
    onChangeByKey?: (params: RetrieveByKeyParams<Query, K>, handler: query.ChangeHandler<E>) => destructor.Destructor;
}
export interface UseListParams<Query extends query.Params, K extends record.Key, E extends record.Keyed<K>> {
    initialQuery?: Query;
    filter?: (item: E) => boolean;
    sort?: compare.Comparator<E>;
    retrieveDebounce?: CrudeTimeSpan;
    useCachedList?: boolean;
}
export interface UseList<Query extends query.Params, K extends record.Key, E extends record.Keyed<K>> {
    (params?: UseListParams<Query, K, E>): UseListReturn<Query, K, E>;
}
export declare const createList: <Query extends query.Params, Key extends record.Key, Data extends record.Keyed<Key>>({ name, retrieve, retrieveByKey, onChange: subscribeToQuery, onChangeByKey, getCached, sort: defaultSort, normalizeQuery, }: CreateListParams<Query, Key, Data>) => UseList<Query, Key, Data>;
export interface UseListItemParams<K extends record.Key, E extends record.Keyed<K>> extends Pick<UseListReturn<query.Params, K, E>, "subscribe" | "getItem"> {
    key: K;
}
export declare const useListItem: <K extends record.Key, E extends record.Keyed<K>>({ key, subscribe, getItem, }: UseListItemParams<K, E>) => E | undefined;
//# sourceMappingURL=list.d.ts.map