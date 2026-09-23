import { type destructor } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { type Callback, type MatchOptions, type Trigger } from "./triggers";
/** Subscribes to every trigger event, returning the unsubscribe. */
export interface Listen {
    (callback: Callback, priority?: number): destructor.Destructor;
}
/** State the {@link Provider} publishes. */
export interface ContextValue {
    listen: Listen;
}
declare const useContext: () => ContextValue;
export { useContext };
/** Props for {@link Provider}. */
export interface ProviderProps extends PropsWithChildren {
    /** Triggers whose browser default is suppressed, such as the browser's own find. */
    preventDefaultOn?: Trigger[];
    preventDefaultOptions?: MatchOptions;
}
/**
 * Listens for keyboard and mouse input on the window and fans it out to every
 * {@link use} subscriber, highest priority first. Mount one near the root of the app.
 * Keystrokes inside a text field reach subscribers only when they carry a modifier.
 */
export declare const Provider: ({ children, preventDefaultOn, preventDefaultOptions, }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map