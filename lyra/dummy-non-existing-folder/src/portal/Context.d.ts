import { type PropsWithChildren, type ReactElement } from "react";
import { Store } from "../store";
/**
 * ContextValue is the registry a {@link Context} shares between portal parts: In parts
 * register their content element under a key, Out parts resolve and subscribe to it.
 */
export interface ContextValue extends Pick<Store.UseKeyedListenersReturn<string>, "subscribe"> {
    /** register makes el resolvable under key, replacing any prior entry. */
    register: (key: string, el: HTMLElement) => void;
    /** unregister removes the entry under key. */
    unregister: (key: string) => void;
    /** get resolves the element registered under key, if any. */
    get: (key: string) => HTMLElement | undefined;
}
declare const useContext: (hookOrComponentName: string) => ContextValue;
export { useContext };
export interface ContextProps extends PropsWithChildren {
}
/**
 * Context owns the key to element registry that links In and Out parts. Every
 * In and the Out parts that host its content must share a Context.
 */
export declare const Context: ({ children }: ContextProps) => ReactElement;
//# sourceMappingURL=Context.d.ts.map