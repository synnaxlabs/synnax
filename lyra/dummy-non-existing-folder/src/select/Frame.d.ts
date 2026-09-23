import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { List } from "../list";
import { type UseMultipleProps, type UseSingleProps } from "./use";
/** Props a custom {@link Frame} trigger receives. */
export interface TriggerProps<K extends record.Key, E extends record.Keyed<K> | undefined> {
    value: K | null;
    /** Reads the entry for a key, subscribing to its changes. */
    useItem: (key: K) => E;
    /** Opens the dialog. */
    onClick: () => void;
}
interface BaseFrameProps<K extends record.Key, E extends record.Keyed<K> | undefined> extends Omit<List.FrameProps<K, E>, "onChange"> {
}
export interface MultipleFrameProps<K extends record.Key, E extends record.Keyed<K> | undefined> extends BaseFrameProps<K, E>, UseMultipleProps<K> {
    multiple: true;
}
export interface SingleFrameProps<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined> extends BaseFrameProps<K, E>, UseSingleProps<K> {
    multiple?: false;
}
/** Props for {@link Frame}. Set `multiple` to switch to a multi-entry selection. */
export type FrameProps<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K>> = MultipleFrameProps<K, E> | SingleFrameProps<K, E>;
export declare function Frame<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K>>({ data, getItem, subscribe, itemHeight, multiple, onFetchMore, overscan, virtual, value, onChange, ...rest }: FrameProps<K, E>): ReactElement;
export declare namespace Frame {
    var displayName: string;
}
export {};
//# sourceMappingURL=Frame.d.ts.map