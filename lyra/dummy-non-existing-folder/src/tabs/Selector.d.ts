import { type ReactElement } from "react";
import { type Component } from "../component";
import { Flex } from "../flex";
import { Haul } from "../haul";
/**
 * The visual variant of a tab selector.
 *
 * - `default`: rounded chip tabs on a borderless strip; the selected tab carries a
 *   subtle fill.
 * - `pill`: separated outlined rounded buttons.
 */
export type Variant = "default" | "pill";
/**
 * Where a tab's contents sit along the strip's main axis. Vertical strips always
 * start-align; a label centered in a tall column reads as adrift.
 */
export type Align = "center" | "start";
/**
 * How tabs claim width.
 *
 * - `elastic`: tabs share the strip and cap at their own label width.
 * - `fixed`: every tab rests at the same standard width.
 * - `content`: every tab rests at its own label width and never compacts.
 *
 * The first two compact toward `--pluto-tabs-tab-min-width` before the strip
 * overflows; `content` overflows straight away.
 */
export type Sizing = "elastic" | "fixed" | "content";
interface ContextValue {
    /** size sets the height of the strip and the typography level of its tabs. */
    size: Component.Size;
    /** variant is the visual variant applied to the strip's tabs. */
    variant: Variant;
}
declare const useContext: (hookOrComponentName: string) => ContextValue;
/** LIST_ROLE is the ARIA role a Selector renders on the tab strip. */
export declare const LIST_ROLE = "tablist";
/** LIST_SELECTOR matches the tab strip rendered by Selector via {@link LIST_ROLE}. */
export declare const LIST_SELECTOR = "[role=\"tablist\"]";
export { useContext as useSelectorContext };
/** The dragging state a strip drop reports, plus the resolved insertion index. */
export interface SelectorOnDropParams extends Haul.OnDropProps {
    /**
     * index is the strip slot the dragged item was dropped at, ranging from 0
     * (before the first tab) to the number of tabs (after the last tab).
     */
    index: number;
}
export interface SelectorProps extends Omit<Flex.BoxProps, "onDrop" | "align"> {
    /** size sets the height of the strip and the typography level of its tabs. */
    size?: Component.Size;
    /**
     * variant is the chassis its tabs wear: borderless chips on a plain strip, or
     * separated outlined pills. Everything else about how the strip behaves is a knob
     * below, which variant only supplies the default for.
     */
    variant?: Variant;
    /** align places a tab's contents along the strip. Defaults per variant. */
    align?: Align;
    /** sizing decides how tabs claim width. Defaults per variant. */
    sizing?: Sizing;
    /**
     * haulType enables drag-and-drop reordering by declaring the Haul item type the
     * strip accepts. When set, dragging an accepted item over the strip opens the slot
     * it would land in and dropping it calls onDrop with the target index. When empty
     * (the default), the strip is passive and registers no drop zone.
     */
    haulType?: string;
    /**
     * canDrop overrides the default acceptance predicate (any item whose type matches
     * haulType). Ignored when haulType is empty.
     */
    canDrop?: Haul.CanDrop;
    /**
     * onDrop fires when an accepted item is dropped on the strip, receiving the
     * dragging state and the resolved insertion index. Return the items the strip
     * consumed, matching the Haul drop contract. Ignored when haulType is empty.
     */
    onDrop?: (params: SelectorOnDropParams) => Haul.Item[];
}
/**
 * Selector is the strip that lays out a Frame's tabs. It renders a tablist with
 * arrow-key roving focus (manual activation: focus moves, Enter or Space selects).
 * Given a haulType it becomes a drag-and-drop target for reordering, owning the
 * insertion geometry and its preview and reporting drops through onDrop.
 */
export declare const Selector: ({ ref, size, variant, align, sizing, haulType, canDrop, onDrop, className, children, direction, x, y, onKeyDown, onDragLeave, empty, gap, ...rest }: SelectorProps) => ReactElement;
//# sourceMappingURL=Selector.d.ts.map