import { type compare, type record, type state } from "@synnaxlabs/x";
import { type FrameProps } from "./Frame";
/** Return value for {@link useStaticData}. Spread it into a {@link Frame}. */
export interface UseStaticDataReturn<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined> extends Required<Pick<FrameProps<K, E>, "getItem">> {
    data: K[];
    retrieve: state.Setter<RetrieveParams, Partial<RetrieveParams>>;
}
/** Query the caller passes to `retrieve` to narrow the data. */
export interface RetrieveParams {
    searchTerm?: string;
    offset?: number;
    limit?: number;
}
export interface UseStaticDataParams<K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>> {
    data: readonly E[];
    /** Drops any item this rejects, before search and sort. */
    filter?: (item: E, params: RetrieveParams) => boolean;
    sort?: compare.Comparator<E>;
}
/**
 * Backs a {@link Frame} with an in-memory array, adding fuzzy search over every field
 * of the entry.
 *
 * @example <List.Frame {...List.useStaticData({ data: MODES })}>
 */
export declare const useStaticData: <K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>>({ data, filter, sort, }: UseStaticDataParams<K, E>) => UseStaticDataReturn<K, E>;
//# sourceMappingURL=useStaticData.d.ts.map