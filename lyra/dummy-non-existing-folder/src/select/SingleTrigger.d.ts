import { type record } from "@synnaxlabs/x";
import { Dialog } from "../dialog";
import { type Icon } from "../icon";
export interface SingleTriggerEntry<K extends record.Key> extends record.KeyedNamed<K> {
    icon?: Icon.ReactElement;
}
/** Props for {@link SingleTrigger}. */
export interface SingleTriggerProps extends Dialog.TriggerProps {
    /** Haul item type this trigger accepts as a drop. Empty accepts nothing. */
    haulType?: string;
    placeholder?: string;
    icon?: Icon.ReactElement;
    /** Whether to render the icon alone, with no name and no caret. */
    iconOnly?: boolean;
    /** Chooses an icon from the selected entry, overriding `icon`. */
    renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
}
/** The button of a {@link Single} selection, showing the selected entry's name. */
export declare const SingleTrigger: <K extends record.Key>({ haulType, placeholder, icon: baseIcon, disabled, iconOnly, hideCaret, renderIcon, preview, ...rest }: SingleTriggerProps) => import("react").JSX.Element;
//# sourceMappingURL=SingleTrigger.d.ts.map