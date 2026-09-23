import { type record } from "@synnaxlabs/x";
import { type FrameProps } from "./Frame";
interface GetItem<K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>> {
    (key?: K): E | undefined;
    (key: K[]): E[];
}
/** Return value for {@link useMapData}. */
export interface UseMapDataReturn<K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>> extends Required<Pick<FrameProps<K, E>, "subscribe">> {
    /** Adds or replaces items, re-rendering only the rows that carry their keys. */
    setItem: (item: E | E[]) => void;
    deleteItem: (key: K | K[]) => void;
    hasItem: (key: K) => boolean;
    getItem: GetItem<K, E>;
}
export interface UseMapDataProps<K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>> {
    initialData?: E[];
}
/**
 * Holds list entries in a ref-backed map and notifies only the rows whose keys change.
 * Use it when entries update often and a re-render of the whole list would be too
 * expensive. It owns the entries, not their order: pass `data` to the frame yourself.
 */
export declare const useMapData: <K extends record.Key = record.Key, E extends record.Keyed<K> = record.Keyed<K>>({ initialData }?: UseMapDataProps<K, E>) => UseMapDataReturn<K, E>;
export {};
//# sourceMappingURL=useMapData.d.ts.map