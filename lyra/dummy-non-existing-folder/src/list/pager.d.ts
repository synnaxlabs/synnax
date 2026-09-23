import { type state } from "@synnaxlabs/x";
/**
 * Parameters for pagination functionality.
 * These parameters are automatically managed by the pager utilities.
 */
export type PagerParams = {
    /** Search term for filtering results */
    searchTerm?: string;
    /** Number of items to skip (for pagination) */
    offset?: number;
    /** Maximum number of items to return per page */
    limit?: number;
};
/** Return type for the usePager hook, providing pagination utilities. */
export interface UsePagerReturn {
    /** Function to fetch the next page of results */
    fetchMore: () => void;
    /** Function to perform a search with the given term */
    search: (term: string) => void;
}
interface RetrieveOptions {
    mode?: "append" | "replace";
}
/** Arguments for the usePager hook. */
export interface UsePagerParams {
    /** Function to retrieve data */
    retrieve: (setter: state.SetArg<PagerParams, Partial<PagerParams>>, options?: RetrieveOptions) => void;
    /** Number of items per page (default: 10) */
    pageSize?: number;
}
/** @returns the params for the page after the given one. */
export declare const page: ({ offset, searchTerm, ...prev }: PagerParams, pageSize?: number) => PagerParams;
/** @returns the given params rewound to the first page of a new search term. */
export declare const search: (prev: PagerParams, searchTerm: string, pageSize?: number) => {
    searchTerm: string;
    offset: number;
    limit: number;
};
/**
 * Turns a flux list query's `retrieve` into paging and search callbacks, tracking the
 * offset itself. Wire `fetchMore` to the frame's `onFetchMore`.
 *
 * @example const { fetchMore, search } = List.usePager({ retrieve, pageSize: 20 });
 */
export declare const usePager: ({ retrieve, pageSize, }: UsePagerParams) => UsePagerReturn;
export {};
//# sourceMappingURL=pager.d.ts.map