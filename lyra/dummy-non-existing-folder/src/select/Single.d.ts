import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Dialog } from "../dialog";
import { type List } from "../list";
import { type DialogProps } from "./Dialog";
import { type SingleFrameProps } from "./Frame";
import { type SingleTriggerProps } from "./SingleTrigger";
export interface SingleProps<K extends record.Key, E extends record.Keyed<K> | undefined> extends Omit<SingleFrameProps<K, E>, "multiple" | "children">, Pick<DialogProps<K>, "emptyContent" | "status" | "onSearch" | "actions" | "footer">, Omit<Dialog.FrameProps, "onChange" | "children" | "variant">, Pick<SingleTriggerProps, "disabled" | "icon" | "haulType">, Pick<List.ItemsProps<K>, "children"> {
    /** Singular name of the thing being selected. It builds the placeholder and the
     * empty and error content. */
    resourceName: string;
    variant?: Dialog.FrameProps["variant"];
    /** Whether to render the trigger flat and inert, for use inside a preview. */
    preview?: boolean;
    triggerProps?: SingleTriggerProps;
    dialogProps?: Dialog.FrameProps;
}
/**
 * A dropdown that selects one entry. Pass `data` and `getItem` from a list data hook,
 * and a `children` render prop for the item.
 *
 * @example
 * <Select.Single resourceName="Channel" value={key} onChange={setKey} {...listProps} />
 */
export declare const Single: <K extends record.Key, E extends record.Keyed<K> | undefined>({ resourceName, onChange, value, allowNone, emptyContent, haulType, data, getItem, subscribe, itemHeight, onFetchMore, disabled, onSearch, status, icon, children, variant, preview, actions, footer, dialogProps, triggerProps, virtual, closeDialogOnSelect, ...rest }: SingleProps<K, E>) => ReactElement;
//# sourceMappingURL=Single.d.ts.map