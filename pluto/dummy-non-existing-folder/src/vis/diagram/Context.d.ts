import { type Viewport } from "../../viewport";
import { diagram } from "./aether";
export interface ContextValue {
    editable: boolean;
    visible: boolean;
    onEditableChange: (v: boolean) => void;
    viewportMode: Viewport.Mode;
    onViewportModeChange: (v: Viewport.Mode) => void;
    fitViewOnResize: boolean;
    setFitViewOnResize: (v: boolean) => void;
    fitViewOptions: diagram.FitViewOptions;
}
export declare const Context: import("react").Context<ContextValue>, useContext: () => ContextValue;
//# sourceMappingURL=Context.d.ts.map