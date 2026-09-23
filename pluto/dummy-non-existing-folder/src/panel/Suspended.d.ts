import { type panel } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
/** useKey resolves the active panel key from the surrounding {@link Suspended}. */
export declare const useKey: (override?: string | undefined) => string;
export declare const useOptionalKey: (override?: string | undefined) => string | undefined;
/** useTabKey resolves the active tab key from the surrounding {@link TabProvider}. */
export declare const useTabKey: (override?: string | undefined) => string;
export declare const useOptionalTabKey: (override?: string | undefined) => string | undefined;
export interface SuspendedProps extends PropsWithChildren {
    panelKey: panel.Key;
}
/**
 * Suspended retrieves the panel into the flux cache and publishes its key to
 * descendants so panel-scoped hooks resolve it without an explicit argument.
 */
export declare const Suspended: ({ panelKey, children }: SuspendedProps) => ReactElement;
export interface TabProviderProps extends PropsWithChildren {
    tabKey: panel.TabKey;
}
/** TabProvider publishes the active tab key to descendants for tab-scoped hooks. */
export declare const TabProvider: ({ tabKey, children }: TabProviderProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map