import { type CrudeTimeSpan, type destructor } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
export interface ContextValue {
    delay: CrudeTimeSpan;
    /** Returns true if a tooltip closed recently enough to skip the open delay. */
    isWarm: () => boolean;
    /** Records a tooltip close, starting the warm window. */
    markClosed: () => void;
    /**
     * Registers the currently open tooltip's close function, closing any other open
     * tooltip. Returns a destructor that deregisters the tooltip.
     */
    acquire: (close: () => void) => destructor.Destructor;
}
export interface ConfigProps extends PropsWithChildren {
    delay?: CrudeTimeSpan;
    skipDelay?: CrudeTimeSpan;
}
declare const useConfig: () => ContextValue;
export { useConfig };
/**
 * Sets the configuration for all tooltips in its children.
 * @param props.delay - The delay before a tooltip opens on hover.
 * @default 700ms.
 * @param props.skipDelay - How long after a tooltip closes a new hover opens instantly.
 * @default 300ms.
 */
export declare const Config: ({ delay, skipDelay, children, }: ConfigProps) => ReactElement;
//# sourceMappingURL=Config.d.ts.map