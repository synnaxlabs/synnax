import { type Theming } from "@synnaxlabs/lyra/theming";
import { type xy } from "@synnaxlabs/x";
import { type FC } from "react";
export interface PreviewProps<C extends object = object> {
    config: C;
    scale?: number;
}
export interface SymbolProps<C extends object = object> extends PreviewProps<C> {
    nodeKey?: string;
    position?: xy.XY;
    selected?: boolean;
    draggable?: boolean;
    onConfigChange?: (data: Partial<C>) => void;
}
export interface Spec<T extends string = string, C extends object = object> {
    key: T;
    name: string;
    Form: FC<{}>;
    Symbol: FC<SymbolProps<C>>;
    defaultConfig: (t: Theming.Theme) => C;
    Preview: FC<PreviewProps<C>>;
    zIndex: number;
}
//# sourceMappingURL=spec.d.ts.map