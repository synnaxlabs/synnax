import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { type UseKeyedListenersReturn } from "./store";
export type MembershipValue<K extends record.Key = record.Key> = K | K[] | undefined;
interface ContextValue<K extends record.Key = record.Key> extends Pick<UseKeyedListenersReturn<K>, "subscribe"> {
    onItem: (key: K) => void;
    setValue: (keys: K[]) => void;
    clear: () => void;
    getValue: () => MembershipValue<K>;
}
export interface MembershipProps<K extends record.Key = record.Key> extends PropsWithChildren, Partial<Pick<ContextValue<K>, "onItem" | "setValue" | "clear">> {
    value: MembershipValue<K>;
}
export interface MembershipItem {
    member: boolean;
    onItem: () => void;
}
/**
 * Membership is a created set-membership context: a provider that publishes a controlled
 * set of keys plus hooks bound to its own React context. Nesting is independent, so a
 * created membership never reads or writes the set of another.
 */
export interface Membership<K extends record.Key = record.Key> {
    /**
     * Context distributes the controlled set to keyed consumers. It diffs value changes
     * and notifies only the listeners whose keys entered or left the set, so item
     * re-renders stay surgical via useItem.
     */
    Context: (props: MembershipProps<K>) => ReactElement;
    useContext: () => ContextValue<K>;
    /**
     * useItem subscribes a single keyed item to the enclosing Provider, re-rendering only
     * when this key's membership flips, and returns a bound onItem action request.
     */
    useItem: (key: K) => MembershipItem;
    /** useIsMember subscribes a single keyed item, re-rendering only when it enters or
     * leaves the set. */
    useIsMember: (key: K) => boolean;
    /** useMembers returns the current members in order. */
    useMembers: () => K[];
    /**
     * useMemberAmong returns the member among the given keys, or undefined when none of
     * them is a member. It subscribes only to the given keys, so consumers stay isolated
     * from changes to the rest of the set. When more than one is a member, the earliest in
     * the set's order wins.
     */
    useMemberAmong: (keys: K[]) => K | undefined;
}
/**
 * createMembership mints a typed set-membership context: a Provider that publishes a
 * controlled set of keys and hooks that read it. Call once per use case so each set
 * carries its own React context and nests independently of other sets.
 * @param name identifies the set in the context display name.
 */
export declare const createMembership: (name: string) => Membership;
export {};
//# sourceMappingURL=membership.d.ts.map