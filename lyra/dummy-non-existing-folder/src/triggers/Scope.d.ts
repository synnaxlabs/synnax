import { type PropsWithChildren, type ReactElement } from "react";
/** Condition is a boolean, or a getter read at the moment a trigger fires. */
export type Condition = boolean | (() => boolean);
/** @returns the value of a {@link Condition}, calling it when it is a getter. */
export declare const resolveCondition: (cond: Condition) => boolean;
declare const useScope: () => () => boolean;
export { useScope };
/** Props for {@link Scope}. */
export interface ScopeProps extends PropsWithChildren {
    active: Condition;
}
/**
 * Scope withholds trigger events from every {@link use} subscriber in its subtree while
 * active resolves false. Scopes nest: an inner scope cannot re-enable triggers that an
 * outer one has switched off. Keep active's identity stable. Every subscriber in the
 * subtree re-renders when it changes, so an inline arrow re-renders all of them on
 * every render of this scope.
 */
export declare const Scope: ({ active, children }: ScopeProps) => ReactElement;
//# sourceMappingURL=Scope.d.ts.map