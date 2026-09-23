import { type ReactElement } from "react";
import { Flex } from "../flex";
export interface ContentProps extends Flex.BoxProps {
    /**
     * itemKey renders this content as the panel for the tab with the same key,
     * mounted only while that tab is selected. When omitted, the content is a
     * plain always-rendered slot for consumers that drive their own visibility,
     * e.g. through portals.
     */
    itemKey?: string;
    /**
     * keepMounted keeps the panel mounted and hidden while its tab is unselected
     * instead of unmounting it. Use for content that is expensive to remount,
     * such as WebGL canvases.
     */
    keepMounted?: boolean;
}
/**
 * Content renders the content area of a composed tabbed interface. See
 * {@link ContentProps.itemKey} for the keyed and keyless modes.
 */
export declare const Content: ({ itemKey, keepMounted, ...rest }: ContentProps) => ReactElement | null;
//# sourceMappingURL=Content.d.ts.map