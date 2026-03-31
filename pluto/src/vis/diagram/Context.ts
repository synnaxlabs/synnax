import { context } from "@/context";
import { type Viewport } from "@/viewport";
import { diagram } from "@/vis/diagram/aether";

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

export const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    editable: true,
    fitViewOnResize: false,
    fitViewOptions: diagram.FIT_VIEW_OPTIONS,
    onEditableChange: () => {},
    onViewportModeChange: () => {},
    setFitViewOnResize: () => {},
    viewportMode: "select",
    visible: true,
  },
  displayName: "Diagram.Context",
});
