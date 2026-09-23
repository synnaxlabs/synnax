import type * as monacoT from "@codingame/monaco-vscode-editor-api";
import { type PropsWithChildren, type ReactElement } from "react";
import { type Language } from "./language";
export type * as Monaco from "@codingame/monaco-vscode-editor-api";
type Monaco = typeof monacoT;
export interface ProviderProps extends PropsWithChildren {
    languages?: Language[];
}
/** Provider makes the Monaco runtime available to descendant editors. It performs no work
 * on mount: monaco initializes lazily on the first editor demand (the first suspending
 * useMonaco call). Languages with a languageServer get a reconnecting client once monaco
 * is ready. */
export declare const Provider: ({ children, languages, }: ProviderProps) => ReactElement;
/** useMonaco returns the initialized Monaco runtime, suspending until it is ready and
 * triggering its lazy initialization on first use. It throws to the nearest error
 * boundary if initialization fails. */
export declare const useMonaco: () => Monaco;
/** useLanguage returns the registered Language descriptor for the given id, or undefined
 * if none is registered. */
export declare const useLanguage: (name: string) => Language | undefined;
//# sourceMappingURL=Provider.d.ts.map