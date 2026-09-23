import { type record } from "@synnaxlabs/x";
import { type Button } from "../button";
import { List } from "../list";
/** Props for {@link ListItem}. */
export type ListItemProps<K extends record.Key = record.Key, E extends Button.ElementType = "div"> = List.ItemProps<K, E>;
/**
 * A {@link List.Item} wired to the enclosing selection, so clicking it selects and its
 * selected and hovered states come from the selection rather than the caller.
 */
export declare const ListItem: <K extends record.Key = record.Key, E extends Button.ElementType = "div">(props: ListItemProps<K, E>) => import("react").JSX.Element;
//# sourceMappingURL=ListItem.d.ts.map