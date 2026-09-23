/** Marks an element as part of the current selection, so a context menu opened over one
 * member acts on all of them. */
export declare const CONTEXT_SELECTED: string;
/** Marks an element as a valid context menu target. */
export declare const CONTEXT_TARGET: string;
/** Class on the rendered context menu itself. */
export declare const CONTEXT_MENU_CLASS: string;
/**
 * Attribute stamped on the context target a menu opened over, for as long as the menu
 * is visible, letting CSS hold the target's hover styling. An attribute rather than a
 * class: React rewrites className on re-render, which would wipe an imperative class.
 */
export declare const CONTEXT_OPEN_ATTRIBUTE = "data-context-menu-open";
//# sourceMappingURL=types.d.ts.map