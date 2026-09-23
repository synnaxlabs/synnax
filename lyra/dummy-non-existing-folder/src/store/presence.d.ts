import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { type UseKeyedListenersReturn } from "./store";
interface ContextValue<K extends record.Key = record.Key> extends Pick<UseKeyedListenersReturn<K>, "subscribe"> {
    getPresent: () => K | undefined;
}
export interface PresenceProps<K extends record.Key = record.Key> extends PropsWithChildren {
    value?: K;
}
/**
 * Presence is a created presence context: a provider that publishes at most one present
 * key plus hooks bound to its own React context. Nesting is independent, so a created
 * presence never reads or writes the presence of another.
 */
export interface Presence<K extends record.Key = record.Key> {
    /**
     * Context distributes the single present key to keyed consumers. On change it notifies
     * only the previous and next holders, so at most two items re-render.
     */
    Context: (props: PresenceProps<K>) => ReactElement;
    useContext: () => ContextValue<K>;
    /**
     * useIsPresent subscribes a single keyed item to the enclosing Provider, re-rendering
     * only when this key gains or loses presence.
     */
    useIsPresent: (key: K) => boolean;
    /** usePresent returns the currently present key, or undefined when none is present. */
    usePresent: () => K | undefined;
}
/**
 * createPresence mints a typed presence context: a Provider that publishes at most one
 * present key and hooks that read it. Presence is a single-valued membership: the
 * hovered item, the active item, the keyboard cursor. Call once per use case so each
 * presence carries its own React context and nests independently of other presences.
 * @param name identifies the presence in the context display name.
 */
export declare const createPresence: (name: string) => Presence;
export {};
//# sourceMappingURL=presence.d.ts.map