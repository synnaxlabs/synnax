import { Theming as Base } from "@synnaxlabs/lyra/theming";
import { type ReactElement } from "react";
/** Props for {@link Provider}. */
export interface ProviderProps extends Base.ProviderProps {
}
/**
 * The lyra theming provider plus a bridge that hands the theme and its fonts to the
 * aether worker thread. Mount it inside the Aether provider.
 */
export declare const Provider: ({ children, ...rest }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map