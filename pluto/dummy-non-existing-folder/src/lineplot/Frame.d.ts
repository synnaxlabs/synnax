import "./Frame.css";
import { type color, type CrudeTimeSpan, type destructor } from "@synnaxlabs/x";
import { type CSSProperties, type DetailedHTMLProps, type HTMLAttributes, type PropsWithChildren, type ReactElement, type Ref } from "react";
import { type z } from "zod";
import { Aether } from "../aether";
import { lineplot } from "./aether";
import { type Viewport } from "../viewport";
import { grid } from "../vis/grid";
type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;
export interface ContextValue {
    id: string;
    setGridEntry: (meta: grid.Region) => void;
    removeGridEntry: (key: string) => void;
    setLine: (meta: LineSpec) => void;
    removeLine: (key: string) => void;
    lines: LineSpec[];
    setViewport: (viewport: Viewport.UseEvent) => void;
    addViewportHandler: (handler: Viewport.UseHandler) => destructor.Destructor;
    setHold: (hold: boolean) => void;
    loading: boolean;
    loadingMessage?: string;
}
declare const useContext: (hookOrComponentName: string) => ContextValue;
export { useContext };
export declare const useViewport: (handle: Viewport.UseHandler, component: string) => void;
export declare const useGridEntry: (meta: grid.Region, component: string) => CSSProperties;
export interface LineSpec {
    key: string;
    legendGroup: string;
    color: color.Crude;
    label: string;
    visible: boolean;
}
/** Ref handle exposed by Frame for imperative access */
export interface FrameRef {
    /** Returns the current bounds for all axes */
    getBounds: () => Promise<lineplot.AxesBounds>;
}
export interface FrameProps extends PropsWithChildren, Partial<Pick<z.input<typeof lineplot.linePlotStateZ>, "clearOverScan" | "hold" | "visible">>, Omit<HTMLDivProps, "ref">, Aether.ComponentProps {
    resizeDebounce?: CrudeTimeSpan;
    onHold?: (hold: boolean) => void;
    loadingMessage?: string;
    ref?: Ref<FrameRef>;
}
export declare const Frame: ({ aetherKey, style, resizeDebounce: debounce, clearOverScan, children, hold, onHold, visible, loadingMessage, ref, ...rest }: FrameProps) => ReactElement;
//# sourceMappingURL=Frame.d.ts.map