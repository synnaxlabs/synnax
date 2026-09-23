import { type PropsWithChildren, type ReactElement } from "react";
import { alamos } from "./aether";
export interface ProviderProps extends PropsWithChildren, alamos.ProviderState {
}
export declare const Provider: ({ children, ...rest }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map