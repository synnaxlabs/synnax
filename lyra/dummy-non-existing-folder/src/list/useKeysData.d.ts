import { type record } from "@synnaxlabs/x";
import { type List } from "./";
export interface UseKeysDataReturn<K extends record.Key = record.Key> extends Required<Pick<List.FrameProps<K, record.Keyed<K>>, "getItem">> {
    data: K[];
}
/**
 * Backs a {@link Frame} with a plain key array, where the key is the whole entry. Use
 * it for a fixed set of options that carry no data of their own.
 */
export declare const useKeysData: <K extends record.Key = record.Key>(data: K[] | readonly K[]) => UseKeysDataReturn<K>;
//# sourceMappingURL=useKeysData.d.ts.map