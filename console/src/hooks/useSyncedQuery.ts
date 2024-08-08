import { Primitive } from "@synnaxlabs/x"

export interface UseSyncedQueryProps<T> {
    key: Primitive[];
    queryFn: () => Promise<T>;
}

export const useSyncedQuery = <T>(
