import "./MultipleTrigger.css";
import { type color, type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";
import { Button } from "../button";
import { type RenderProp } from "../component/renderProp";
import { Haul } from "../haul";
import { Icon } from "../icon";
import { Tag } from "../tag";
export interface MultipleEntry<K extends record.Key> extends record.KeyedNamed<K> {
    icon?: Icon.ReactElement;
    color?: color.Crude;
    alias?: string;
}
export interface MultipleTagProps<K extends record.Key> extends Omit<Tag.TagProps, "onDragStart"> {
    itemKey: K;
    onDragStart: (key: K) => void;
    renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
}
/** Props for {@link MultipleTrigger}. */
export interface MultipleTriggerProps<K extends record.Key, E extends record.Keyed<K> | undefined = MultipleEntry<K> | undefined> extends Pick<Button.ButtonProps, "variant" | "disabled" | "preview"> {
    /** Haul item type this trigger accepts as a drop. Empty accepts nothing. */
    haulType?: string;
    /** Builds the haul item for an entry dragged out of the trigger. */
    createHaulItem?: (entry: NonNullable<E>) => Haul.Item;
    placeholder?: ReactNode;
    icon?: Icon.ReactElement;
    /** Whether to show only a count instead of one tag per entry. */
    hideTags?: boolean;
    children?: RenderProp<MultipleTagProps<K>>;
    renderIcon?: (entry: unknown) => Icon.ReactElement | undefined;
}
/** @returns whether a drag carries at least one entry of the type not already selected. */
export declare const staticCanDrop: <K extends record.Key>({ items: entities }: Haul.DraggingState, haulType: string, value: K[] | readonly K[], disabled?: boolean) => boolean;
/** The button of a {@link Multiple} selection, showing one removable tag per entry. */
export declare const MultipleTrigger: <K extends record.Key, E extends record.Keyed<K> | undefined = MultipleEntry<K> | undefined>({ haulType, createHaulItem, disabled, placeholder, variant, preview, icon, hideTags, children, renderIcon, }: MultipleTriggerProps<K, E>) => ReactElement | null;
//# sourceMappingURL=MultipleTrigger.d.ts.map