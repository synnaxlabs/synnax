import { type location, type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, type RefCallback } from "react";
/** Function interface for getting items from a list by key(s). */
export interface GetItem<K extends record.Key, E extends record.Keyed<K> | undefined> extends GetSingleItem<K, E>, GetMultipleItems<K, E> {
}
/** Reads one item by key. */
export interface GetSingleItem<K extends record.Key, E extends record.Keyed<K> | undefined> {
    (key: K): E | undefined;
}
/** Reads many items at once, dropping any key with no item. */
export interface GetMultipleItems<K extends record.Key, E extends record.Keyed<K> | undefined> {
    (keys: K[]): E[];
}
/** Joins a single-key and a multi-key reader into one {@link GetItem}. */
export declare const createGetItem: <K extends record.Key, E extends record.Keyed<K> | undefined>(first: GetSingleItem<K, E>, second: GetMultipleItems<K, E>) => GetItem<K, E>;
/** One item the enclosing frame asks its children to render. */
export interface ItemSpec<K extends record.Key = record.Key> {
    key: K;
    index: number;
    /** Pixel offset from the top of the list, set only when virtualized. */
    translate?: number;
}
export interface DataContextValue<K extends record.Key = record.Key> {
    data: K[];
    getItems: () => ItemSpec<K>[];
    getTotalSize: () => number | undefined;
    sentinelRef?: RefCallback<HTMLDivElement>;
}
export interface UtilContextValue<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined> {
    ref: RefCallback<HTMLDivElement | null>;
    getItem?: GetItem<K, E>;
    subscribe?: (callback: () => void, key: K) => () => void;
    scrollToIndex: (index: number, direction?: location.Y) => void;
    itemHeight?: number;
}
export declare const useUtilContext: <K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined>() => UtilContextValue<K, E>;
/** Props for {@link Frame}. A data hook such as `useStaticData` supplies most of them. */
export interface FrameProps<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined> extends PropsWithChildren, Pick<UtilContextValue<K, E>, "getItem" | "subscribe"> {
    /** The keys to render, in order. */
    data: K[];
    /** Whether to render only the visible window. Needed above a few hundred items. */
    virtual?: boolean;
    /** Extra items to render past each edge of the visible window. */
    overscan?: number;
    /** Row height in pixels. Virtualization estimates from it. */
    itemHeight?: number;
    /** Called when the list scrolls near its end. */
    onFetchMore?: () => void;
}
/** @returns a scroller for the enclosing {@link Frame}, stable as the list scrolls. */
export declare const useScroller: <K extends record.Key = record.Key>() => Pick<UtilContextValue<K>, "scrollToIndex">;
/**
 * useItemHeight returns the row height the enclosing Frame was given. It reads the
 * util context, which does not change as the list scrolls.
 */
export declare const useItemHeight: () => number | undefined;
/**
 * Reads the item for a key from the enclosing {@link Frame} and re-renders the caller
 * when that one item changes. Use it inside a list item, so the list does not re-render
 * on every entry update.
 */
export declare const useItem: <K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined>(key: K) => E | undefined;
/**
 * Reads the full state of the enclosing {@link Frame}: the keys, the visible window,
 * and the item readers. It re-renders on every scroll, so prefer {@link useItem} inside
 * an item.
 */
export declare const useData: <K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined>() => DataContextValue<K> & UtilContextValue<K, E>;
/** {@link Frame} before memoization. Prefer `Frame`. */
export declare const BaseFrame: <K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined>({ virtual, ...rest }: FrameProps<K, E>) => ReactElement;
/**
 * Holds the data for a list and hands it to its children through context. It renders no
 * element of its own: pair it with {@link Items} for the scroll container.
 *
 * @example
 * <List.Frame {...List.useStaticData({ data })}>
 *   <List.Items>{(p) => <List.Item {...p} />}</List.Items>
 * </List.Frame>
 */
export declare const Frame: typeof BaseFrame;
//# sourceMappingURL=Frame.d.ts.map