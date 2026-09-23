import "./Provider.css";
import { type Component } from "@synnaxlabs/lyra/component";
import { type ReactElement } from "react";
import { Aether } from "../../aether";
import { range } from "./aether";
export interface ProviderProps extends Aether.ComponentProps {
    visible?: boolean;
    onHasAnnotationsChange?: (hasAnnotations: boolean) => void;
    menu?: Component.RenderProp<range.SelectedState>;
}
export declare const Provider: ({ aetherKey, menu, visible, onHasAnnotationsChange, ...rest }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map