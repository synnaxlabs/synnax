import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Dialog as BaseDialog } from "../dialog";
import { type List } from "../list";
import { type DialogProps } from "./Dialog";
import { type MultipleFrameProps } from "./Frame";
import { type MultipleTriggerProps } from "./MultipleTrigger";
export interface MultipleProps<K extends record.Key, E extends record.Keyed<K> | undefined> extends Omit<MultipleFrameProps<K, E>, "multiple" | "children">, Pick<DialogProps<K>, "emptyContent" | "status" | "onSearch" | "actions" | "footer">, Omit<BaseDialog.FrameProps, "onChange" | "children" | "variant">, Pick<MultipleTriggerProps<K, E>, "disabled" | "icon" | "haulType" | "createHaulItem">, Pick<List.ItemsProps<K>, "children"> {
    /** Singular name of the thing being selected. It is pluralized for the placeholder. */
    resourceName: string;
    /** Renders one tag in the trigger. Defaults to the entry name. */
    renderTag?: MultipleTriggerProps<K, E>["children"];
    triggerProps?: MultipleTriggerProps<K, E>;
    dialogProps?: BaseDialog.FrameProps;
    variant?: BaseDialog.FrameProps["variant"];
    preview?: boolean;
}
/**
 * A dropdown that selects any number of entries, showing each as a removable tag in the
 * trigger. Shift extends a range and control toggles one entry.
 */
export declare const Multiple: <K extends record.Key, E extends record.Keyed<K> | undefined>({ resourceName, value, onChange, data, getItem, subscribe, haulType, createHaulItem, icon, disabled, onSearch, emptyContent, status, onFetchMore, children, renderTag, actions, footer, allowNone, replaceOnSingle, triggerProps, virtual, dialogProps, variant, preview, ...rest }: MultipleProps<K, E>) => ReactElement;
//# sourceMappingURL=Multiple.d.ts.map