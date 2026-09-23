import { type state } from "@synnaxlabs/lyra/state";
import { color } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { z } from "zod";
export declare const relevancyZ: z.ZodObject<{
    lastUsed: z.ZodNumber;
    count: z.ZodNumber;
    relevance: z.ZodNumber;
}, z.core.$strip>;
export declare const frequentZ: z.ZodRecord<z.ZodString, z.ZodObject<{
    lastUsed: z.ZodNumber;
    count: z.ZodNumber;
    relevance: z.ZodNumber;
}, z.core.$strip>>;
export interface Frequent extends z.infer<typeof frequentZ> {
}
export declare const contextStateZ: z.ZodObject<{
    frequent: z.ZodRecord<z.ZodString, z.ZodObject<{
        lastUsed: z.ZodNumber;
        count: z.ZodNumber;
        relevance: z.ZodNumber;
    }, z.core.$strip>>;
}, z.core.$strip>;
export interface ContextState extends z.infer<typeof contextStateZ> {
}
export interface ContextValue extends ContextState {
    updateFrequent: (color: color.Color) => void;
}
export declare const ZERO_CONTEXT_STATE: ContextState;
declare const useContext: () => ContextValue;
export { useContext };
export declare const recalculate: (limit: number, frequent: Frequent) => Frequent;
export interface ProviderProps extends PropsWithChildren<{}> {
    useState?: state.PureUse<ContextState>;
}
export declare const Provider: ({ useState, children, }: ProviderProps) => ReactElement;
export declare const useFrequent: () => color.Color[];
export declare const useFrequentUpdater: () => ((color: color.Color) => void);
//# sourceMappingURL=Provider.d.ts.map