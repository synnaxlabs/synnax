import "./Dialog.css";
import { type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";
import { type z } from "zod";
import { Dialog as BaseDialog } from "../dialog";
import { List } from "../list";
import { type SearchInputProps } from "./SearchInput";
import { Status } from "../status";
/** Props for {@link Dialog}. */
export interface DialogProps<K extends record.Key> extends Omit<BaseDialog.DialogProps, "children">, Omit<SearchInputProps, "searchPlaceholder">, Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
    status?: Status.Status<z.ZodNever>;
    resourceName: string;
    /** Pinned below the scrollable list; stays visible regardless of list length. */
    footer?: ReactNode;
}
/** Props for the content shown when a selection has nothing to offer. */
export interface DefaultEmptyContentProps extends Status.SummaryProps {
    resourceName: string;
}
declare const Base: {
    <K extends record.Key>({ onSearch, children, emptyContent, status, resourceName, actions, footer, className, ...rest }: DialogProps<K>): ReactElement;
    displayName: string;
};
/**
 * The dropdown of a selection: its search field, its list, and its empty and error
 * content. It sizes the list to whole rows, so its growth animates.
 */
export declare const Dialog: typeof Base;
export {};
//# sourceMappingURL=Dialog.d.ts.map