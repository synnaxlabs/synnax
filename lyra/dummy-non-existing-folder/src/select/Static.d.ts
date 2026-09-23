import { type optional, type record } from "@synnaxlabs/x";
import { type Icon } from "../icon";
import { List } from "../list";
import { type SingleProps } from "./Single";
/** One option in a {@link Static} selection. */
export interface StaticEntry<K extends record.Key> extends record.KeyedNamed<K> {
    icon?: Icon.ReactElement;
}
export interface StaticProps<K extends record.Key, E extends StaticEntry<K> = StaticEntry<K>> extends optional.Optional<Omit<SingleProps<K, E>, "data" | "getItem" | "subscribe">, "children">, List.UseStaticDataParams<K, E> {
}
/**
 * A {@link Single} over an in-memory array, with client-side search and a default item
 * that renders the entry icon and name.
 *
 * @example <Select.Static resourceName="Mode" data={MODES} value={v} onChange={set} />
 */
export declare const Static: <K extends record.Key, E extends record.KeyedNamed<K>>({ data, filter, children, virtual, ...rest }: StaticProps<K, E>) => import("react").JSX.Element;
//# sourceMappingURL=Static.d.ts.map