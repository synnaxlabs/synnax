import "./Mosaic.css";
import { panel } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, type ReactNode } from "react";
import { Mosaic as Base } from "../mosaic";
export interface MosaicProps extends Omit<Base.FrameProps, "onDrop" | "onCreate" | "onResize" | "onSelect" | "children" | "contextMenu"> {
    selected?: panel.TabKey[];
    onSelect?: (tabKey: panel.TabKey) => void;
    children: Component.RenderProp<{}>;
    tabName?: Component.RenderProp<{}>;
    onCreateTab?: () => panel.NewTab | undefined;
    resolveDroppedTab?: (key: string) => panel.NewTab | undefined;
    /** Renders the full tab context menu. When a tab is under the cursor, the
     * render prop runs inside that tab's scope, so {@link CloseTabMenuItem} and
     * {@link SplitTabMenuItems} can be composed with caller items in any order. */
    contextMenu?: Component.RenderProp<Menu.ContextMenuMenuProps>;
    /** Rendered in a leaf's content area when the leaf has no tabs. */
    emptyContent?: ReactNode;
    /** Tab the mosaic collapses to: the frame renders a single leaf holding this
     * tab in place of the tree. Content stays mounted through the portal layer;
     * background tabs detach from the DOM, so their canvas draws stop. */
    overlaid?: panel.TabKey;
    /** Called when the user requests to exit the overlaid state. */
    onStopOverlay?: () => void;
}
/** Creates a new tab. Bound by the embedding app; shown on the create button. */
export declare const CREATE_TAB_TRIGGER: Triggers.Trigger;
/** Enters a tab's overlaid (focused) state. Escape is the way out, so the exit
 * affordances hint that instead. Bound by the embedding app. */
export declare const OVERLAY_TRIGGER: Triggers.Trigger;
/** Closes the focused tab. Bound by the embedding app; shown on the close menu item. */
export declare const CLOSE_TRIGGER: Triggers.Trigger;
/** Props for {@link CloseTabMenuItem}. */
export interface CloseTabMenuItemProps {
    /** Shows the {@link CLOSE_TRIGGER} hint. The app binds the trigger and chooses the
     * tab it acts on, so only the app knows whether this tab is that tab. */
    triggerIndicator?: boolean;
}
/** CloseTabMenuItem closes the context menu's tab. Hidden from a viewer who cannot
 * write the panel. Must render inside the tab context menu passed to {@link Mosaic}. */
export declare const CloseTabMenuItem: ({ triggerIndicator, }?: CloseTabMenuItemProps) => ReactElement | null;
/** SplitTabMenuItems splits the context menu's tab horizontally or vertically.
 * Hidden when the tab cannot be split, or from a viewer who cannot write the panel.
 * Must render inside the tab context menu passed to {@link Mosaic}. */
export declare const SplitTabMenuItems: () => ReactElement | null;
export declare const Mosaic: ({ selected, onSelect, children, tabName, onCreateTab, resolveDroppedTab, contextMenu, emptyContent, overlaid, onStopOverlay, className, ...rest }: MosaicProps) => ReactElement | null;
//# sourceMappingURL=Mosaic.d.ts.map