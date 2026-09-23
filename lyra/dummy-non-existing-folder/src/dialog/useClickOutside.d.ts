import { type RefObject } from "react";
/**
 * Marks an element as the logical parent of a portaled subtree. Set it on the element
 * a portal renders from, and set {@link PORTAL_OWNER_ATTR} to the same value on the
 * portaled element.
 */
export declare const PORTAL_ID_ATTR = "data-portal-id";
/** Points a portaled element at the {@link PORTAL_ID_ATTR} of its logical parent. */
export declare const PORTAL_OWNER_ATTR = "data-portal-owner";
/** Props for {@link useClickOutside}. */
export interface UseClickOutsideProps {
    ref: RefObject<HTMLElement | null>;
    /** Elements, or a predicate over the event, whose clicks do not count as outside. */
    exclude?: Array<RefObject<HTMLElement>> | ((e: MouseEvent) => boolean);
    onClickOutside: () => void;
}
/**
 * Calls back on a click outside the element, treating a portaled subtree as inside the
 * element it renders from. Clicks past the viewport edge, such as on a scrollbar, do
 * not count.
 */
export declare const useClickOutside: ({ ref, onClickOutside, exclude, }: UseClickOutsideProps) => void;
//# sourceMappingURL=useClickOutside.d.ts.map