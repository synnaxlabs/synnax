import { type record } from "@synnaxlabs/x";
import { type FrameProps } from "./Frame";
export interface UseCombinedDataParams<K extends record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined> {
    first: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
    second: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
}
/**
 * Concatenates two list data sources into one, `first` before `second`. Reads and
 * subscriptions hit both.
 */
export declare const useCombinedData: <K extends record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined>({ first, second, }: UseCombinedDataParams<K, E>) => FrameProps<K, E>;
//# sourceMappingURL=useCombinedData.d.ts.map